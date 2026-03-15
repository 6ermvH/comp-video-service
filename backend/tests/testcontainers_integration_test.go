//go:build integration

package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestTestcontainersPostgresWithMigrations(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}

	ctx := context.Background()
	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("compvideo"),
		postgres.WithUsername("cvs"),
		postgres.WithPassword("cvs_secret"),
	)
	if err != nil {
		t.Skipf("docker/testcontainers unavailable: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer db.Close()

	for i := 0; i < 30; i++ {
		if err := db.Ping(ctx); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	root := os.Getenv("PROJECT_ROOT")
	if root == "" {
		root = "/home/g-feskov/work/comp-video-service"
	}
	m1, err := os.ReadFile(filepath.Join(root, "backend", "migrations", "001_init.up.sql"))
	if err != nil {
		t.Fatalf("read migration 001: %v", err)
	}
	m2, err := os.ReadFile(filepath.Join(root, "backend", "migrations", "002_study_schema.up.sql"))
	if err != nil {
		t.Fatalf("read migration 002: %v", err)
	}

	if _, err := db.Exec(ctx, string(m1)); err != nil {
		t.Fatalf("exec migration 001: %v", err)
	}
	if _, err := db.Exec(ctx, string(m2)); err != nil {
		t.Fatalf("exec migration 002: %v", err)
	}

	var exists bool
	err = db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema='public' AND table_name='responses'
		)`).Scan(&exists)
	if err != nil {
		t.Fatalf("check table: %v", err)
	}
	if !exists {
		t.Fatal("expected responses table to exist")
	}
}

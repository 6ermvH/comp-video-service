//go:build integration

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	testDBOnce sync.Once
	testDBPool *pgxpool.Pool
	testDBErr  error
	testPg     *postgres.PostgresContainer
)

func TestMain(m *testing.M) {
	code := m.Run()
	if testDBPool != nil {
		testDBPool.Close()
	}
	if testPg != nil {
		_ = testPg.Terminate(context.Background())
	}
	os.Exit(code)
}

func mustOpenDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	testDBOnce.Do(func() {
		ctx := context.Background()
		testPg, testDBErr = postgres.Run(
			ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("compvideo"),
			postgres.WithUsername("cvs"),
			postgres.WithPassword("cvs_secret"),
		)
		if testDBErr != nil {
			return
		}

		dsn, err := testPg.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			testDBErr = err
			return
		}

		testDBPool, testDBErr = pgxpool.New(ctx, dsn)
		if testDBErr != nil {
			return
		}

		for i := 0; i < 30; i++ {
			if err := testDBPool.Ping(ctx); err == nil {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}

		if err := applyMigrations(ctx, testDBPool); err != nil {
			testDBErr = err
			return
		}
	})

	if testDBErr != nil {
		t.Fatalf("init test db: %v", testDBErr)
	}
	return testDBPool
}

func applyMigrations(ctx context.Context, db *pgxpool.Pool) error {
	m1, err := readMigrationFile("001_init.up.sql")
	if err != nil {
		return err
	}
	m2, err := readMigrationFile("002_study_schema.up.sql")
	if err != nil {
		return err
	}
	if _, err := db.Exec(ctx, m1); err != nil {
		return fmt.Errorf("exec migration 001: %w", err)
	}
	if _, err := db.Exec(ctx, m2); err != nil {
		return fmt.Errorf("exec migration 002: %w", err)
	}
	return nil
}

func readMigrationFile(filename string) (string, error) {
	candidates := []string{
		filepath.Join("migrations", filename),
		filepath.Join("..", "migrations", filename),
		filepath.Join("backend", "migrations", filename),
		filepath.Join("..", "backend", "migrations", filename),
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("migration file not found: %s", filename)
}

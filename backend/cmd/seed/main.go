// cmd/seed/main.go — creates the first admin account and a demo study.
//
// Usage:
//   DATABASE_URL="postgres://..." go run ./cmd/seed \
//     -username admin -password secret
//
// Or set SEED_USERNAME / SEED_PASSWORD env vars.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	username := flag.String("username", env("SEED_USERNAME", "admin"), "admin username")
	password := flag.String("password", env("SEED_PASSWORD", ""), "admin password (required)")
	flag.Parse()

	if *password == "" {
		log.Fatal("password is required: use -password flag or SEED_PASSWORD env var")
	}

	dsn := env("DATABASE_URL", "")
	if dsn == "" {
		log.Fatal("DATABASE_URL env var is not set")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt error: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer func() {
		_ = conn.Close(ctx)
	}()

	// 1. Create admin
	_, err = conn.Exec(ctx,
		`INSERT INTO admins (username, password_hash)
		 VALUES ($1, $2)
		 ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		*username, string(hash),
	)
	if err != nil {
		log.Fatalf("insert admin: %v", err)
	}

	fmt.Printf("✓ Admin '%s' created/updated successfully\n", *username)

	// 2. Create sample studies if none exist
	var studyCount int
	err = conn.QueryRow(ctx, "SELECT count(*) FROM studies").Scan(&studyCount)
	if err == nil && studyCount == 0 {
		var studyID string
		err = conn.QueryRow(ctx,
			`INSERT INTO studies (name, effect_type, status) 
			 VALUES ('Demo Flooding Study', 'flooding', 'active') RETURNING id`,
		).Scan(&studyID)

		if err == nil {
			// Create a couple of demo groups
			if _, err := conn.Exec(ctx, `INSERT INTO groups (study_id, name) VALUES ($1, 'Group A')`, studyID); err != nil {
				log.Fatalf("insert group A: %v", err)
			}
			if _, err := conn.Exec(ctx, `INSERT INTO groups (study_id, name) VALUES ($1, 'Group B')`, studyID); err != nil {
				log.Fatalf("insert group B: %v", err)
			}
			fmt.Printf("✓ Sample study 'Demo Flooding Study' created\n")
		}
	} else {
		fmt.Printf("✓ Studies already exist, skipped seeding studies\n")
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

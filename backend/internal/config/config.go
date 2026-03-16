package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	Port string

	// PostgreSQL
	DatabaseURL string

	// Default admin bootstrap
	DefaultAdminUsername string
	DefaultAdminPassword string

	// S3 / MinIO
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3PublicURL       string // public presigned URL base (shown to browser)
	S3UsePathStyle    bool

	// JWT
	JWTSecret string

	// CORS
	CORSOrigins []string

	// Migrations
	MigrationsPath string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getenv("PORT", "8080"),
		DatabaseURL:          getenv("DATABASE_URL", "postgres://cvs:cvs_secret@localhost:5432/compvideo"),
		DefaultAdminUsername: getenv("DEFAULT_ADMIN_USERNAME", "admin"),
		DefaultAdminPassword: getenv("DEFAULT_ADMIN_PASSWORD", "admin_123_videogen"),
		S3Endpoint:           getenv("S3_ENDPOINT", "http://minio:9000"),
		S3Region:             getenv("S3_REGION", "us-east-1"),
		S3Bucket:             getenv("S3_BUCKET", "videos"),
		S3AccessKeyID:        getenv("MINIO_ROOT_USER", "minioadmin"),
		S3SecretAccessKey:    getenv("MINIO_ROOT_PASSWORD", "minioadmin"),
		S3PublicURL:          getenv("S3_PUBLIC_URL", "http://localhost:9000"),
		S3UsePathStyle:       getenv("S3_USE_PATH_STYLE", "true") == "true",
		JWTSecret:            getenv("JWT_SECRET", ""),
		CORSOrigins:          splitComma(getenv("CORS_ORIGINS", "http://localhost:5173")),
		MigrationsPath:       getenv("MIGRATIONS_PATH", "file://migrations"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

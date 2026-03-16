package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// AdminRepository handles admin account DB operations.
type AdminRepository struct {
	db *pgxpool.Pool
}

// NewAdminRepository creates a new AdminRepository.
func NewAdminRepository(db *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{db: db}
}

// GetByUsername returns an admin by username.
func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	q := `SELECT id, username, password_hash, created_at
		  FROM admins WHERE username = $1`
	var a model.Admin
	err := r.db.QueryRow(ctx, q, username).Scan(
		&a.ID, &a.Username, &a.PasswordHash, &a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get admin by username: %w", err)
	}
	return &a, nil
}

// Upsert replaces or creates the admin credentials for the given username.
func (r *AdminRepository) Upsert(ctx context.Context, username, passwordHash string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO admins (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username)
		DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		username, passwordHash,
	)
	if err != nil {
		return fmt.Errorf("upsert admin: %w", err)
	}
	return nil
}

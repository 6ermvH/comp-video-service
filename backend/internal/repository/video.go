package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// VideoRepository handles all video asset DB operations.
type VideoRepository struct {
	db *pgxpool.Pool
}

// NewVideoRepository creates a new VideoRepository.
func NewVideoRepository(db *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{db: db}
}

// Create inserts a new video asset and returns created record.
func (r *VideoRepository) Create(ctx context.Context, tx pgxTx, v *model.Video) (*model.Video, error) {
	q := `
		INSERT INTO video_assets (
			source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at`

	row := queryRow(r.db, tx, q,
		v.SourceItemID, v.MethodType, v.Title, v.Description, v.S3Key, v.DurationMS,
		v.Status, v.Width, v.Height, v.FPS, v.Codec, v.Checksum,
	)

	return scanVideo(row)
}

// LinkOrCreate inserts a new video asset or links an existing one by s3_key.
func (r *VideoRepository) LinkOrCreate(ctx context.Context, v *model.Video) (*model.Video, error) {
	q := `
		INSERT INTO video_assets (
			source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (s3_key) DO UPDATE
			SET source_item_id = EXCLUDED.source_item_id,
			    method_type    = EXCLUDED.method_type,
			    updated_at     = now()
		RETURNING id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at`

	row := r.db.QueryRow(ctx, q,
		v.SourceItemID, v.MethodType, v.Title, v.Description, v.S3Key, v.DurationMS,
		v.Status, v.Width, v.Height, v.FPS, v.Codec, v.Checksum,
	)
	return scanVideo(row)
}

// GetByID returns a video asset by UUID.
func (r *VideoRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Video, error) {
	q := `
		SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at
		FROM video_assets WHERE id = $1`
	row := r.db.QueryRow(ctx, q, id)
	return scanVideo(row)
}

// ListBySourceItem returns both assets for one source item.
func (r *VideoRepository) ListBySourceItem(ctx context.Context, sourceItemID uuid.UUID) ([]*model.Video, error) {
	q := `
		SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at
		FROM video_assets
		WHERE source_item_id = $1 AND status = 'active'
		ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, q, sourceItemID)
	if err != nil {
		return nil, fmt.Errorf("list by source item: %w", err)
	}
	defer rows.Close()

	var out []*model.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListActive returns all active video assets.
func (r *VideoRepository) ListActive(ctx context.Context) ([]*model.Video, error) {
	q := `
		SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at
		FROM video_assets WHERE status = 'active' ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list active videos: %w", err)
	}
	defer rows.Close()

	var out []*model.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Archive sets the video asset status to archived.
func (r *VideoRepository) Archive(ctx context.Context, tx pgxTx, id uuid.UUID) error {
	_, err := execQuery(r.db, tx,
		`UPDATE video_assets SET status='archived', updated_at=now() WHERE id=$1`, id)
	return err
}

// scanner is shared by repositories in this package.
type scanner interface {
	Scan(dest ...any) error
}

func scanVideo(row scanner) (*model.Video, error) {
	var v model.Video
	err := row.Scan(
		&v.ID, &v.SourceItemID, &v.MethodType, &v.Title, &v.Description, &v.S3Key,
		&v.DurationMS, &v.Status, &v.Width, &v.Height, &v.FPS, &v.Codec,
		&v.Checksum, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan video: %w", err)
	}
	return &v, nil
}

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

// Link sets source_item_id and method_type on an existing video asset.
func (r *VideoRepository) Link(ctx context.Context, videoID uuid.UUID, sourceItemID uuid.UUID, methodType string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE video_assets SET source_item_id = $2, method_type = $3, updated_at = now() WHERE id = $1`,
		videoID, sourceItemID, methodType,
	)
	return err
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

// ListAll returns all video assets ordered by creation date desc.
func (r *VideoRepository) ListAll(ctx context.Context) ([]*model.Video, error) {
	q := `
		SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at
		FROM video_assets
		ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list all videos: %w", err)
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

// ListPaged returns a page of video assets and the total count.
// If search is non-empty, results are filtered by title using case-insensitive ILIKE.
func (r *VideoRepository) ListPaged(ctx context.Context, page, perPage int, search string) ([]*model.Video, int, error) {
	offset := (page - 1) * perPage

	if search != "" {
		pattern := "%" + search + "%"
		var total int
		if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM video_assets WHERE title ILIKE $1`, pattern).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count videos: %w", err)
		}
		q := `
			SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
				status, width, height, fps, codec, checksum, created_at, updated_at
			FROM video_assets
			WHERE title ILIKE $3
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`
		rows, err := r.db.Query(ctx, q, perPage, offset, pattern)
		if err != nil {
			return nil, 0, fmt.Errorf("list paged videos: %w", err)
		}
		defer rows.Close()
		var out []*model.Video
		for rows.Next() {
			v, err := scanVideo(rows)
			if err != nil {
				return nil, 0, err
			}
			out = append(out, v)
		}
		if out == nil {
			out = make([]*model.Video, 0)
		}
		return out, total, rows.Err()
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM video_assets`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count videos: %w", err)
	}
	q := `
		SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at
		FROM video_assets
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list paged videos: %w", err)
	}
	defer rows.Close()

	var out []*model.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	if out == nil {
		out = make([]*model.Video, 0)
	}
	return out, total, rows.Err()
}

// ListFree returns all video assets not linked to any source_item (free for pairing).
func (r *VideoRepository) ListFree(ctx context.Context) ([]*model.Video, error) {
	q := `
		SELECT id, source_item_id, method_type, title, description, s3_key, duration_ms,
			status, width, height, fps, codec, checksum, created_at, updated_at
		FROM video_assets
		WHERE source_item_id IS NULL
		ORDER BY method_type, title`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list free videos: %w", err)
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
	if out == nil {
		out = make([]*model.Video, 0)
	}
	return out, rows.Err()
}

// Delete removes a video asset.
// Returns false if blocked: video is still linked to a source_item or referenced in pair_presentations.
func (r *VideoRepository) Delete(ctx context.Context, id uuid.UUID) (deleted bool, err error) {
	var sourceItemID *uuid.UUID
	if err := r.db.QueryRow(ctx,
		`SELECT source_item_id FROM video_assets WHERE id = $1`, id,
	).Scan(&sourceItemID); err != nil {
		return false, fmt.Errorf("find asset: %w", err)
	}
	if sourceItemID != nil {
		return false, nil
	}

	var count int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM pair_presentations WHERE left_asset_id = $1 OR right_asset_id = $1`, id,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check pair_presentations: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	if _, err := r.db.Exec(ctx, `DELETE FROM video_assets WHERE id = $1`, id); err != nil {
		return false, fmt.Errorf("delete video asset: %w", err)
	}
	return true, nil
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

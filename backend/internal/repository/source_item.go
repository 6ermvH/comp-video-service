package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// SourceItemRepository provides source item queries.
type SourceItemRepository struct {
	db *pgxpool.Pool
}

func NewSourceItemRepository(db *pgxpool.Pool) *SourceItemRepository {
	return &SourceItemRepository{db: db}
}

func (r *SourceItemRepository) Create(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO source_items (
			study_id, group_id, source_image_id, pair_code, difficulty, is_attention_check, notes
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, study_id, group_id, source_image_id, pair_code, difficulty,
			is_attention_check, notes, created_at`,
		item.StudyID, item.GroupID, item.SourceImageID, item.PairCode,
		item.Difficulty, item.IsAttentionCheck, item.Notes,
	)
	return scanSourceItem(row)
}

func (r *SourceItemRepository) ListByStudy(ctx context.Context, studyID uuid.UUID) ([]*model.SourceItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, study_id, group_id, source_image_id, pair_code, difficulty,
			is_attention_check, notes, created_at
		FROM source_items
		WHERE study_id = $1
		ORDER BY created_at ASC`, studyID)
	if err != nil {
		return nil, fmt.Errorf("list source items: %w", err)
	}
	defer rows.Close()

	out := make([]*model.SourceItem, 0)
	for rows.Next() {
		si, err := scanSourceItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, si)
	}
	return out, rows.Err()
}

func (r *SourceItemRepository) ListWithFilters(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItem, error) {
	q := `
		SELECT id, study_id, group_id, source_image_id, pair_code, difficulty,
			is_attention_check, notes, created_at
		FROM source_items WHERE 1=1`
	args := make([]any, 0, 2)
	idx := 1
	if studyID != nil {
		q += fmt.Sprintf(" AND study_id = $%d", idx)
		args = append(args, *studyID)
		idx++
	}
	if groupID != nil {
		q += fmt.Sprintf(" AND group_id = $%d", idx)
		args = append(args, *groupID)
	}
	q += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list source items with filters: %w", err)
	}
	defer rows.Close()

	out := make([]*model.SourceItem, 0)
	for rows.Next() {
		si, err := scanSourceItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, si)
	}
	return out, rows.Err()
}

func (r *SourceItemRepository) ListWithDetails(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItemDetail, error) {
	q := `
		SELECT
			si.id, si.study_id, si.group_id, g.name AS group_name,
			si.source_image_id, si.pair_code, si.difficulty,
			si.is_attention_check, si.notes, si.created_at,
			(SELECT COUNT(*) FROM video_assets va WHERE va.source_item_id = si.id) AS asset_count,
			(SELECT COUNT(*) FROM responses r
			 JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
			 WHERE pp.source_item_id = si.id) AS response_count
		FROM source_items si
		JOIN groups g ON g.id = si.group_id
		WHERE 1=1`
	args := make([]any, 0, 2)
	idx := 1
	if studyID != nil {
		q += fmt.Sprintf(" AND si.study_id = $%d", idx)
		args = append(args, *studyID)
		idx++
	}
	if groupID != nil {
		q += fmt.Sprintf(" AND si.group_id = $%d", idx)
		args = append(args, *groupID)
	}
	q += " ORDER BY si.created_at ASC"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list source items with details: %w", err)
	}
	defer rows.Close()

	out := make([]*model.SourceItemDetail, 0)
	for rows.Next() {
		var d model.SourceItemDetail
		if err := rows.Scan(
			&d.ID, &d.StudyID, &d.GroupID, &d.GroupName,
			&d.SourceImageID, &d.PairCode, &d.Difficulty,
			&d.IsAttentionCheck, &d.Notes, &d.CreatedAt,
			&d.AssetCount, &d.ResponseCount,
		); err != nil {
			return nil, fmt.Errorf("scan source item detail: %w", err)
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

// UpdateAttentionCheck sets the is_attention_check flag for the given source item.
func (r *SourceItemRepository) UpdateAttentionCheck(ctx context.Context, id uuid.UUID, isAttentionCheck bool) error {
	if _, err := r.db.Exec(ctx,
		`UPDATE source_items SET is_attention_check = $2 WHERE id = $1`,
		id, isAttentionCheck,
	); err != nil {
		return fmt.Errorf("update attention check: %w", err)
	}
	return nil
}

// GetByIDWithDetails returns a single source item with enriched detail fields.
func (r *SourceItemRepository) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*model.SourceItemDetail, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			si.id, si.study_id, si.group_id, g.name AS group_name,
			si.source_image_id, si.pair_code, si.difficulty,
			si.is_attention_check, si.notes, si.created_at,
			(SELECT COUNT(*) FROM video_assets va WHERE va.source_item_id = si.id) AS asset_count,
			(SELECT COUNT(*) FROM responses r
			 JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
			 WHERE pp.source_item_id = si.id) AS response_count
		FROM source_items si
		JOIN groups g ON g.id = si.group_id
		WHERE si.id = $1`, id)

	var d model.SourceItemDetail
	if err := row.Scan(
		&d.ID, &d.StudyID, &d.GroupID, &d.GroupName,
		&d.SourceImageID, &d.PairCode, &d.Difficulty,
		&d.IsAttentionCheck, &d.Notes, &d.CreatedAt,
		&d.AssetCount, &d.ResponseCount,
	); err != nil {
		return nil, fmt.Errorf("get source item by id: %w", err)
	}
	return &d, nil
}

// Delete removes a source item and unlinks its video assets (sets source_item_id = NULL).
// Returns false if the source item has responses — deletion is blocked.
func (r *SourceItemRepository) Delete(ctx context.Context, id uuid.UUID) (deleted bool, err error) {
	var count int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM responses r
		JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
		WHERE pp.source_item_id = $1`, id,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check responses: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	if _, err := r.db.Exec(ctx,
		`UPDATE video_assets SET source_item_id = NULL, updated_at = now() WHERE source_item_id = $1`, id,
	); err != nil {
		return false, fmt.Errorf("unlink video assets: %w", err)
	}

	if _, err := r.db.Exec(ctx,
		`DELETE FROM pair_presentations WHERE source_item_id = $1`, id,
	); err != nil {
		return false, fmt.Errorf("delete pair_presentations: %w", err)
	}

	if _, err := r.db.Exec(ctx,
		`DELETE FROM source_items WHERE id = $1`, id,
	); err != nil {
		return false, fmt.Errorf("delete source_item: %w", err)
	}

	return true, nil
}

// ResponseCountsByStudy returns response counts per source item in a study.
func (r *SourceItemRepository) ResponseCountsByStudy(ctx context.Context, studyID uuid.UUID) (map[uuid.UUID]int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			si.id,
			COUNT(r.id) AS response_count
		FROM source_items si
		LEFT JOIN pair_presentations pp ON pp.source_item_id = si.id
		LEFT JOIN responses r ON r.pair_presentation_id = pp.id
		WHERE si.study_id = $1
		GROUP BY si.id`, studyID)
	if err != nil {
		return nil, fmt.Errorf("response counts by study: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]int64)
	for rows.Next() {
		var sourceItemID uuid.UUID
		var cnt int64
		if err := rows.Scan(&sourceItemID, &cnt); err != nil {
			return nil, fmt.Errorf("scan response counts by study: %w", err)
		}
		out[sourceItemID] = cnt
	}
	return out, rows.Err()
}

func scanSourceItem(row scanner) (*model.SourceItem, error) {
	var s model.SourceItem
	err := row.Scan(
		&s.ID, &s.StudyID, &s.GroupID, &s.SourceImageID, &s.PairCode,
		&s.Difficulty, &s.IsAttentionCheck, &s.Notes, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan source item: %w", err)
	}
	return &s, nil
}

package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// PairPresentationRepository stores assigned tasks.
type PairPresentationRepository struct {
	db *pgxpool.Pool
}

func NewPairPresentationRepository(db *pgxpool.Pool) *PairPresentationRepository {
	return &PairPresentationRepository{db: db}
}

func (r *PairPresentationRepository) Create(ctx context.Context, pp *model.PairPresentation) (*model.PairPresentation, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO pair_presentations (
			participant_id, source_item_id, left_asset_id, right_asset_id,
			left_method_type, right_method_type, task_order, is_attention_check, is_practice
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, participant_id, source_item_id, left_asset_id, right_asset_id,
			left_method_type, right_method_type, task_order, is_attention_check, is_practice, created_at`,
		pp.ParticipantID, pp.SourceItemID, pp.LeftAssetID, pp.RightAssetID,
		pp.LeftMethodType, pp.RightMethodType, pp.TaskOrder, pp.IsAttentionCheck, pp.IsPractice,
	)
	return scanPairPresentation(row)
}

// GetNextPendingByToken returns first un-answered task for participant.
func (r *PairPresentationRepository) GetNextPendingByToken(ctx context.Context, sessionToken string) (*model.PairPresentation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT pp.id, pp.participant_id, pp.source_item_id, pp.left_asset_id, pp.right_asset_id,
			pp.left_method_type, pp.right_method_type, pp.task_order, pp.is_attention_check,
			pp.is_practice, pp.created_at
		FROM pair_presentations pp
		JOIN participants p ON p.id = pp.participant_id
		LEFT JOIN responses r ON r.pair_presentation_id = pp.id AND r.participant_id = p.id
		WHERE p.session_token = $1 AND r.id IS NULL
		ORDER BY pp.task_order ASC
		LIMIT 1`, sessionToken)
	return scanPairPresentation(row)
}

func (r *PairPresentationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PairPresentation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, participant_id, source_item_id, left_asset_id, right_asset_id,
			left_method_type, right_method_type, task_order, is_attention_check,
			is_practice, created_at
		FROM pair_presentations
		WHERE id = $1`, id)
	return scanPairPresentation(row)
}

func (r *PairPresentationRepository) CountByParticipant(ctx context.Context, participantID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM pair_presentations WHERE participant_id = $1`, participantID).Scan(&total)
	return total, err
}

func scanPairPresentation(row scanner) (*model.PairPresentation, error) {
	var p model.PairPresentation
	err := row.Scan(
		&p.ID, &p.ParticipantID, &p.SourceItemID, &p.LeftAssetID, &p.RightAssetID,
		&p.LeftMethodType, &p.RightMethodType, &p.TaskOrder,
		&p.IsAttentionCheck, &p.IsPractice, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan pair presentation: %w", err)
	}
	return &p, nil
}

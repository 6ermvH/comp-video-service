package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// ResponseRepository handles respondent responses and analytics stats.
type ResponseRepository struct {
	db *pgxpool.Pool
}

func NewResponseRepository(db *pgxpool.Pool) *ResponseRepository {
	return &ResponseRepository{db: db}
}

func (r *ResponseRepository) Create(ctx context.Context, resp *model.Response) (*model.Response, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO responses (
			participant_id, pair_presentation_id, choice, reason_codes,
			confidence, response_time_ms, replay_count, custom_reason
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, participant_id, pair_presentation_id, choice, reason_codes,
			confidence, response_time_ms, replay_count, custom_reason, created_at`,
		resp.ParticipantID, resp.PairPresentationID, resp.Choice,
		resp.ReasonCodes, resp.Confidence, resp.ResponseTimeMS, resp.ReplayCount,
		resp.CustomReason,
	)
	return scanResponse(row)
}

func (r *ResponseRepository) CountByStudy(ctx context.Context, studyID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM responses r
		JOIN participants p ON p.id = r.participant_id
		WHERE p.study_id = $1`, studyID).Scan(&count)
	return count, err
}

func (r *ResponseRepository) CountChoicesByStudy(ctx context.Context, studyID uuid.UUID) (left int64, right int64, tie int64, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN r.choice = 'left' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN r.choice = 'right' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN r.choice = 'tie' THEN 1 ELSE 0 END), 0)
		FROM responses r
		JOIN participants p ON p.id = r.participant_id
		WHERE p.study_id = $1`, studyID).Scan(&left, &right, &tie)
	return
}

func (r *ResponseRepository) CountTotal(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM responses`).Scan(&count)
	return count, err
}

func (r *ResponseRepository) CountFastResponses(ctx context.Context, thresholdMS int) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM responses
		WHERE response_time_ms IS NOT NULL AND response_time_ms < $1`, thresholdMS).Scan(&count)
	return count, err
}

func (r *ResponseRepository) StraightLiningParticipants(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		WITH participant_pref AS (
			SELECT participant_id,
				GREATEST(
					SUM(CASE WHEN choice = 'left' THEN 1 ELSE 0 END),
					SUM(CASE WHEN choice = 'right' THEN 1 ELSE 0 END),
					SUM(CASE WHEN choice = 'tie' THEN 1 ELSE 0 END)
				) AS top_choice,
				COUNT(*) AS total
			FROM responses
			GROUP BY participant_id
		)
		SELECT COUNT(*)
		FROM participant_pref
		WHERE total >= 5 AND top_choice::float / total::float >= 0.9`).Scan(&count)
	return count, err
}

func (r *ResponseRepository) CountByParticipant(ctx context.Context, participantID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT count(*) FROM responses WHERE participant_id = $1`, participantID).Scan(&count)
	return count, err
}

func (r *ResponseRepository) CountFastByParticipant(ctx context.Context, participantID uuid.UUID, thresholdMS int) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM responses
		WHERE participant_id = $1
		  AND response_time_ms IS NOT NULL
		  AND response_time_ms < $2`, participantID, thresholdMS).Scan(&count)
	return count, err
}

// AttentionCheckStats returns (total attention-check responses, failed checks).
func (r *ResponseRepository) AttentionCheckStats(ctx context.Context, participantID uuid.UUID) (int64, int64, error) {
	var total, failed int64
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COALESCE(SUM(
				CASE
					WHEN (pp.left_asset_id = pp.right_asset_id OR pp.left_method_type = pp.right_method_type)
					     AND r.choice <> 'tie'
					THEN 1 ELSE 0
				END
			), 0) AS failed
		FROM responses r
		JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
		WHERE r.participant_id = $1
		  AND pp.is_attention_check = true`, participantID).Scan(&total, &failed)
	if err != nil {
		return 0, 0, err
	}
	return total, failed, nil
}

// CountAttentionCheckFailures returns the total number of responses where the
// participant chose the candidate side in an attention-check pair (baseline is
// always the correct answer in such pairs, so candidate selection = failure).
func (r *ResponseRepository) CountAttentionCheckFailures(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM responses r
		JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
		WHERE pp.is_attention_check = true
		  AND (
		    (pp.left_method_type = 'candidate' AND r.choice = 'left')
		    OR (pp.right_method_type = 'candidate' AND r.choice = 'right')
		  )`).Scan(&count)
	return count, err
}

func scanResponse(row scanner) (*model.Response, error) {
	var resp model.Response
	err := row.Scan(
		&resp.ID, &resp.ParticipantID, &resp.PairPresentationID, &resp.Choice,
		&resp.ReasonCodes, &resp.Confidence, &resp.ResponseTimeMS, &resp.ReplayCount,
		&resp.CustomReason, &resp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan response: %w", err)
	}
	return &resp, nil
}

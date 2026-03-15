package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// InteractionLogRepository stores low-level client events.
type InteractionLogRepository struct {
	db *pgxpool.Pool
}

func NewInteractionLogRepository(db *pgxpool.Pool) *InteractionLogRepository {
	return &InteractionLogRepository{db: db}
}

func (r *InteractionLogRepository) Create(ctx context.Context, event *model.InteractionLog) (*model.InteractionLog, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO interaction_logs (
			participant_id, pair_presentation_id, event_type, payload_json
		)
		VALUES ($1,$2,$3,$4)
		RETURNING id, participant_id, pair_presentation_id, event_type, event_ts, payload_json`,
		event.ParticipantID, event.PairPresentationID, event.EventType, event.PayloadJSON,
	)

	var out model.InteractionLog
	err := row.Scan(
		&out.ID, &out.ParticipantID, &out.PairPresentationID,
		&out.EventType, &out.EventTS, &out.PayloadJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("create interaction log: %w", err)
	}
	return &out, nil
}

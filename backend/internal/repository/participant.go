package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// ParticipantRepository manages respondent sessions.
type ParticipantRepository struct {
	db *pgxpool.Pool
}

func NewParticipantRepository(db *pgxpool.Pool) *ParticipantRepository {
	return &ParticipantRepository{db: db}
}

func (r *ParticipantRepository) Create(ctx context.Context, p *model.Participant) (*model.Participant, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO participants (session_token, study_id, device_type, browser, role, experience)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, session_token, study_id, device_type, browser, role, experience,
			started_at, completed_at, quality_flag`,
		p.SessionToken, p.StudyID, p.DeviceType, p.Browser, p.Role, p.Experience,
	)
	return scanParticipant(row)
}

func (r *ParticipantRepository) GetByToken(ctx context.Context, token string) (*model.Participant, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, session_token, study_id, device_type, browser, role, experience,
			started_at, completed_at, quality_flag
		FROM participants WHERE session_token = $1`, token)
	return scanParticipant(row)
}

func (r *ParticipantRepository) CompleteByToken(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE participants SET completed_at = now()
		WHERE session_token = $1`, token)
	return err
}

func (r *ParticipantRepository) UpdateQualityFlag(ctx context.Context, participantID uuid.UUID, qualityFlag string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE participants SET quality_flag = $2
		WHERE id = $1`, participantID, qualityFlag)
	return err
}

func scanParticipant(row scanner) (*model.Participant, error) {
	var p model.Participant
	err := row.Scan(
		&p.ID, &p.SessionToken, &p.StudyID, &p.DeviceType, &p.Browser,
		&p.Role, &p.Experience, &p.StartedAt, &p.CompletedAt, &p.QualityFlag,
	)
	if err != nil {
		return nil, fmt.Errorf("scan participant: %w", err)
	}
	return &p, nil
}

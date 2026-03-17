package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// StudyRepository handles study CRUD.
type StudyRepository struct {
	db *pgxpool.Pool
}

func NewStudyRepository(db *pgxpool.Pool) *StudyRepository {
	return &StudyRepository{db: db}
}

func (r *StudyRepository) Create(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
	tie := true
	reasons := true
	confidence := true
	if req.TieOptionEnabled != nil {
		tie = *req.TieOptionEnabled
	}
	if req.ReasonsEnabled != nil {
		reasons = *req.ReasonsEnabled
	}
	if req.ConfidenceEnabled != nil {
		confidence = *req.ConfidenceEnabled
	}
	maxTasks := req.MaxTasksPerParticipant
	if maxTasks <= 0 {
		maxTasks = 20
	}
	var instructions *string
	if req.InstructionsText != "" {
		instructions = &req.InstructionsText
	}

	q := `
		INSERT INTO studies (
			name, effect_type, status, max_tasks_per_participant,
			instructions_text, tie_option_enabled, reasons_enabled, confidence_enabled
		)
		VALUES ($1,$2,'draft',$3,$4,$5,$6,$7)
		RETURNING id, name, effect_type, status, max_tasks_per_participant,
			instructions_text, tie_option_enabled, reasons_enabled, confidence_enabled, created_at`

	row := r.db.QueryRow(ctx, q,
		req.Name, req.EffectType, maxTasks, instructions,
		tie, reasons, confidence,
	)
	return scanStudy(row)
}

func (r *StudyRepository) List(ctx context.Context) ([]*model.Study, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, effect_type, status, max_tasks_per_participant,
			instructions_text, tie_option_enabled, reasons_enabled, confidence_enabled, created_at
		FROM studies ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list studies: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Study, 0)
	for rows.Next() {
		s, err := scanStudy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *StudyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Study, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, effect_type, status, max_tasks_per_participant,
			instructions_text, tie_option_enabled, reasons_enabled, confidence_enabled, created_at
		FROM studies WHERE id = $1`, id)
	return scanStudy(row)
}

func (r *StudyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Study, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE studies SET status = $2
		WHERE id = $1
		RETURNING id, name, effect_type, status, max_tasks_per_participant,
			instructions_text, tie_option_enabled, reasons_enabled, confidence_enabled, created_at`,
		id, status,
	)
	return scanStudy(row)
}

// Update applies a partial update to a study.
func (r *StudyRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE studies SET
			status                    = COALESCE($2, status),
			name                      = COALESCE($3, name),
			effect_type               = COALESCE($4, effect_type),
			max_tasks_per_participant = COALESCE($5, max_tasks_per_participant),
			instructions_text         = COALESCE($6, instructions_text),
			tie_option_enabled        = COALESCE($7, tie_option_enabled),
			reasons_enabled           = COALESCE($8, reasons_enabled),
			confidence_enabled        = COALESCE($9, confidence_enabled)
		WHERE id = $1
		RETURNING id, name, effect_type, status, max_tasks_per_participant,
			instructions_text, tie_option_enabled, reasons_enabled, confidence_enabled, created_at`,
		id,
		req.Status,
		req.Name,
		req.EffectType,
		req.MaxTasksPerParticipant,
		req.InstructionsText,
		req.TieOptionEnabled,
		req.ReasonsEnabled,
		req.ConfidenceEnabled,
	)
	return scanStudy(row)
}

func scanStudy(row scanner) (*model.Study, error) {
	var s model.Study
	err := row.Scan(
		&s.ID, &s.Name, &s.EffectType, &s.Status, &s.MaxTasksPerParticipant,
		&s.InstructionsText, &s.TieOptionEnabled, &s.ReasonsEnabled, &s.ConfidenceEnabled, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan study: %w", err)
	}
	return &s, nil
}

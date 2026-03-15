package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
)

// GroupRepository handles groups in a study.
type GroupRepository struct {
	db *pgxpool.Pool
}

func NewGroupRepository(db *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Create(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
	targetVotes := req.TargetVotesPerPair
	if targetVotes <= 0 {
		targetVotes = 10
	}
	row := r.db.QueryRow(ctx, `
		INSERT INTO groups (study_id, name, description, priority, target_votes_per_pair)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, study_id, name, description, priority, target_votes_per_pair, created_at`,
		studyID, req.Name, req.Description, req.Priority, targetVotes,
	)
	return scanGroup(row)
}

func (r *GroupRepository) ListByStudy(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, study_id, name, description, priority, target_votes_per_pair, created_at
		FROM groups
		WHERE study_id = $1
		ORDER BY priority ASC, created_at ASC`, studyID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	out := make([]*model.Group, 0)
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func scanGroup(row scanner) (*model.Group, error) {
	var g model.Group
	err := row.Scan(&g.ID, &g.StudyID, &g.Name, &g.Description, &g.Priority, &g.TargetVotesPerPair, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan group: %w", err)
	}
	return &g, nil
}

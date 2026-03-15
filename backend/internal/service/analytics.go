package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/repository"
)

// EffectAnalytics is an effect-level aggregation.
type EffectAnalytics struct {
	EffectType string  `json:"effect_type"`
	Responses  int64   `json:"responses"`
	TieRate    float64 `json:"tie_rate"`
}

// GroupAnalytics is a group-level aggregation.
type GroupAnalytics struct {
	GroupID   uuid.UUID `json:"group_id"`
	GroupName string    `json:"group_name"`
	Responses int64     `json:"responses"`
	TieRate   float64   `json:"tie_rate"`
}

// AnalyticsOverview is global analytics summary.
type AnalyticsOverview struct {
	TotalResponses    int64              `json:"total_responses"`
	TotalParticipants int64              `json:"total_participants"`
	TieRate           float64            `json:"tie_rate"`
	Effects           []*EffectAnalytics `json:"effects"`
	Groups            []*GroupAnalytics  `json:"groups"`
}

// StudyAnalytics is per-study breakdown.
type StudyAnalytics struct {
	StudyID      uuid.UUID         `json:"study_id"`
	Total        int64             `json:"total"`
	LeftWins     int64             `json:"left_wins"`
	RightWins    int64             `json:"right_wins"`
	TieCount     int64             `json:"tie_count"`
	LeftWinRate  float64           `json:"left_win_rate"`
	RightWinRate float64           `json:"right_win_rate"`
	TieRate      float64           `json:"tie_rate"`
	Groups       []*GroupAnalytics `json:"groups"`
}

// AnalyticsService provides analytics endpoints.
type AnalyticsService struct {
	db           *pgxpool.Pool
	responseRepo *repository.ResponseRepository
}

func NewAnalyticsService(db *pgxpool.Pool, responseRepo *repository.ResponseRepository) *AnalyticsService {
	return &AnalyticsService{db: db, responseRepo: responseRepo}
}

func (s *AnalyticsService) Overview(ctx context.Context) (*AnalyticsOverview, error) {
	total, err := s.responseRepo.CountTotal(ctx)
	if err != nil {
		return nil, err
	}

	var participants int64
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM participants`).Scan(&participants); err != nil {
		return nil, err
	}

	var tie int64
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN choice = 'tie' THEN 1 ELSE 0 END), 0)
		FROM responses`).Scan(&tie); err != nil {
		return nil, err
	}

	tieRate := 0.0
	if total > 0 {
		tieRate = float64(tie) / float64(total)
	}

	effects, err := s.queryEffectAnalytics(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.queryGroupAnalytics(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &AnalyticsOverview{
		TotalResponses:    total,
		TotalParticipants: participants,
		TieRate:           tieRate,
		Effects:           effects,
		Groups:            groups,
	}, nil
}

func (s *AnalyticsService) StudyDetail(ctx context.Context, studyID uuid.UUID) (*StudyAnalytics, error) {
	left, right, tie, err := s.responseRepo.CountChoicesByStudy(ctx, studyID)
	if err != nil {
		return nil, err
	}
	total := left + right + tie

	out := &StudyAnalytics{
		StudyID:   studyID,
		Total:     total,
		LeftWins:  left,
		RightWins: right,
		TieCount:  tie,
	}
	if total > 0 {
		out.LeftWinRate = float64(left) / float64(total)
		out.RightWinRate = float64(right) / float64(total)
		out.TieRate = float64(tie) / float64(total)
	}

	groups, err := s.queryGroupAnalytics(ctx, &studyID)
	if err != nil {
		return nil, err
	}
	out.Groups = groups

	return out, nil
}

func (s *AnalyticsService) queryEffectAnalytics(ctx context.Context) ([]*EffectAnalytics, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			st.effect_type,
			COUNT(*) AS responses,
			COALESCE(SUM(CASE WHEN r.choice = 'tie' THEN 1 ELSE 0 END), 0)::float / COUNT(*)::float AS tie_rate
		FROM responses r
		JOIN participants p ON p.id = r.participant_id
		JOIN studies st ON st.id = p.study_id
		GROUP BY st.effect_type
		ORDER BY responses DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*EffectAnalytics, 0)
	for rows.Next() {
		var e EffectAnalytics
		if err := rows.Scan(&e.EffectType, &e.Responses, &e.TieRate); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (s *AnalyticsService) queryGroupAnalytics(ctx context.Context, studyID *uuid.UUID) ([]*GroupAnalytics, error) {
	query := `
		SELECT
			g.id,
			g.name,
			COUNT(r.id) AS responses,
			COALESCE(SUM(CASE WHEN r.choice = 'tie' THEN 1 ELSE 0 END), 0)::float / NULLIF(COUNT(r.id), 0)::float AS tie_rate
		FROM groups g
		JOIN source_items si ON si.group_id = g.id
		LEFT JOIN pair_presentations pp ON pp.source_item_id = si.id
		LEFT JOIN responses r ON r.pair_presentation_id = pp.id`

	args := make([]any, 0, 1)
	if studyID != nil {
		query += ` WHERE g.study_id = $1`
		args = append(args, *studyID)
	}
	query += ` GROUP BY g.id, g.name ORDER BY responses DESC, g.name ASC`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*GroupAnalytics, 0)
	for rows.Next() {
		var (
			g       GroupAnalytics
			tieRate *float64
		)
		if err := rows.Scan(&g.GroupID, &g.GroupName, &g.Responses, &tieRate); err != nil {
			return nil, err
		}
		if tieRate != nil {
			g.TieRate = *tieRate
		}
		out = append(out, &g)
	}
	return out, rows.Err()
}

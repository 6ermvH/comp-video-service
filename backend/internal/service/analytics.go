package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EffectAnalytics is an effect-level aggregation.
type EffectAnalytics struct {
	EffectType       string  `json:"effect_type"`
	Responses        int64   `json:"responses"`
	TieRate          float64 `json:"tie_rate"`
	CandidateWinRate float64 `json:"candidate_win_rate"`
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
	TotalSourceItems  int64              `json:"total_source_items"`
	TieRate           float64            `json:"tie_rate"`
	CandidateWinRate  float64            `json:"candidate_win_rate"`
	CompletionRate    float64            `json:"completion_rate"`
	Effects           []*EffectAnalytics `json:"effects"`
	Groups            []*GroupAnalytics  `json:"groups"`
}

// PairStat is per-source-item analytics for one study.
type PairStat struct {
	SourceItemID     uuid.UUID `json:"source_item_id"`
	PairCode         *string   `json:"pair_code"`
	Difficulty       *string   `json:"difficulty"`
	GroupName        string    `json:"group_name"`
	TotalResponses   int64     `json:"total_responses"`
	CandidateWins    int64     `json:"candidate_wins"`
	BaselineWins     int64     `json:"baseline_wins"`
	TieCount         int64     `json:"tie_count"`
	CandidateWinRate float64   `json:"candidate_win_rate"`
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
	db           analyticsDB
	responseRepo analyticsResponseRepository
}

type analyticsResponseRepository interface {
	CountTotal(ctx context.Context) (int64, error)
	CountChoicesByStudy(ctx context.Context, studyID uuid.UUID) (left int64, right int64, tie int64, err error)
}

type analyticsRow interface {
	Scan(dest ...any) error
}

type analyticsRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type analyticsDB interface {
	QueryRow(ctx context.Context, sql string, args ...any) analyticsRow
	Query(ctx context.Context, sql string, args ...any) (analyticsRows, error)
}

type pgxAnalyticsDB struct {
	pool *pgxpool.Pool
}

type pgxRowAdapter struct {
	row pgx.Row
}

func (r pgxRowAdapter) Scan(dest ...any) error { return r.row.Scan(dest...) }

type pgxRowsAdapter struct {
	rows pgx.Rows
}

func (r pgxRowsAdapter) Next() bool             { return r.rows.Next() }
func (r pgxRowsAdapter) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r pgxRowsAdapter) Err() error             { return r.rows.Err() }
func (r pgxRowsAdapter) Close()                 { r.rows.Close() }

func (d pgxAnalyticsDB) QueryRow(ctx context.Context, sql string, args ...any) analyticsRow {
	return pgxRowAdapter{row: d.pool.QueryRow(ctx, sql, args...)}
}

func (d pgxAnalyticsDB) Query(ctx context.Context, sql string, args ...any) (analyticsRows, error) {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgxRowsAdapter{rows: rows}, nil
}

//go:generate go run go.uber.org/mock/mockgen -source=analytics.go -destination=analytics_mocks_test.go -package=service

func NewAnalyticsService(db *pgxpool.Pool, responseRepo analyticsResponseRepository) *AnalyticsService {
	return &AnalyticsService{db: pgxAnalyticsDB{pool: db}, responseRepo: responseRepo}
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

	var totalSourceItems int64
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM source_items`).Scan(&totalSourceItems); err != nil {
		return nil, err
	}

	var tie int64
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN choice = 'tie' THEN 1 ELSE 0 END), 0)
		FROM responses`).Scan(&tie); err != nil {
		return nil, err
	}

	var candidateWins int64
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(COUNT(*), 0)
		FROM responses r
		JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
		WHERE (pp.left_method_type = 'candidate' AND r.choice = 'left')
		   OR (pp.right_method_type = 'candidate' AND r.choice = 'right')`).Scan(&candidateWins); err != nil {
		return nil, err
	}

	var completed int64
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM participants WHERE completed_at IS NOT NULL`).Scan(&completed); err != nil {
		return nil, err
	}

	tieRate := 0.0
	if total > 0 {
		tieRate = float64(tie) / float64(total)
	}
	candidateWinRate := 0.0
	if total > 0 {
		candidateWinRate = float64(candidateWins) / float64(total)
	}
	completionRate := 0.0
	if participants > 0 {
		completionRate = float64(completed) / float64(participants)
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
		TotalSourceItems:  totalSourceItems,
		TieRate:           tieRate,
		CandidateWinRate:  candidateWinRate,
		CompletionRate:    completionRate,
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

func (s *AnalyticsService) PairBreakdown(ctx context.Context, studyID uuid.UUID) ([]*PairStat, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			si.id,
			si.pair_code,
			si.difficulty,
			g.name AS group_name,
			COUNT(r.id) AS total_responses,
			COALESCE(SUM(CASE
				WHEN (pp.left_method_type = 'candidate' AND r.choice = 'left')
				  OR (pp.right_method_type = 'candidate' AND r.choice = 'right')
				THEN 1 ELSE 0
			END), 0) AS candidate_wins,
			COALESCE(SUM(CASE
				WHEN (pp.left_method_type = 'baseline' AND r.choice = 'left')
				  OR (pp.right_method_type = 'baseline' AND r.choice = 'right')
				THEN 1 ELSE 0
			END), 0) AS baseline_wins,
			COALESCE(SUM(CASE WHEN r.choice = 'tie' THEN 1 ELSE 0 END), 0) AS tie_count
		FROM source_items si
		JOIN groups g ON g.id = si.group_id
		LEFT JOIN pair_presentations pp ON pp.source_item_id = si.id
		LEFT JOIN responses r ON r.pair_presentation_id = pp.id
		WHERE si.study_id = $1
		  AND si.is_attention_check = false
		GROUP BY si.id, si.pair_code, si.difficulty, g.name
		ORDER BY total_responses DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*PairStat, 0)
	for rows.Next() {
		var stat PairStat
		if err := rows.Scan(
			&stat.SourceItemID,
			&stat.PairCode,
			&stat.Difficulty,
			&stat.GroupName,
			&stat.TotalResponses,
			&stat.CandidateWins,
			&stat.BaselineWins,
			&stat.TieCount,
		); err != nil {
			return nil, err
		}
		if stat.TotalResponses > 0 {
			stat.CandidateWinRate = float64(stat.CandidateWins) / float64(stat.TotalResponses)
		}
		out = append(out, &stat)
	}
	return out, rows.Err()
}

func (s *AnalyticsService) queryEffectAnalytics(ctx context.Context) ([]*EffectAnalytics, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			st.effect_type,
			COUNT(*) AS responses,
			COALESCE(SUM(CASE WHEN r.choice = 'tie' THEN 1 ELSE 0 END), 0)::float / COUNT(*)::float AS tie_rate,
			COALESCE(SUM(CASE
				WHEN (pp.left_method_type = 'candidate' AND r.choice = 'left')
				  OR (pp.right_method_type = 'candidate' AND r.choice = 'right')
				THEN 1 ELSE 0
			END), 0)::float / COUNT(*)::float AS candidate_win_rate
		FROM responses r
		JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
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
		if err := rows.Scan(&e.EffectType, &e.Responses, &e.TieRate, &e.CandidateWinRate); err != nil {
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

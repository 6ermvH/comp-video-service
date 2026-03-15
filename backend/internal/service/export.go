package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExportService produces CSV and JSON snapshots for admin.
type ExportService struct {
	db exportDB
}

type exportRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type exportDB interface {
	Query(ctx context.Context, sql string, args ...any) (exportRows, error)
}

type pgxExportRows struct {
	rows pgx.Rows
}

func (r pgxExportRows) Next() bool             { return r.rows.Next() }
func (r pgxExportRows) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r pgxExportRows) Err() error             { return r.rows.Err() }
func (r pgxExportRows) Close()                 { r.rows.Close() }

type pgxExportDB struct {
	db *pgxpool.Pool
}

func (d pgxExportDB) Query(ctx context.Context, sql string, args ...any) (exportRows, error) {
	rows, err := d.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgxExportRows{rows: rows}, nil
}

//go:generate go run go.uber.org/mock/mockgen -source=export.go -destination=export_mocks_test.go -package=service

func NewExportService(db *pgxpool.Pool) *ExportService {
	return &ExportService{db: pgxExportDB{db: db}}
}

func (s *ExportService) ExportCSV(ctx context.Context) ([]byte, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			r.id,
			r.participant_id,
			p.session_token,
			p.study_id,
			r.pair_presentation_id,
			pp.source_item_id,
			pp.task_order,
			r.choice,
			COALESCE(array_to_string(r.reason_codes, '|'), ''),
			COALESCE(r.confidence::text, ''),
			COALESCE(r.response_time_ms::text, ''),
			r.replay_count,
			pp.is_attention_check,
			r.created_at
		FROM responses r
		JOIN participants p ON p.id = r.participant_id
		JOIN pair_presentations pp ON pp.id = r.pair_presentation_id
		ORDER BY r.created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("export csv query: %w", err)
	}
	defer rows.Close()

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	headers := []string{
		"response_id", "participant_id", "session_token", "study_id", "pair_presentation_id",
		"source_item_id", "task_order",
		"choice", "reason_codes", "confidence", "response_time_ms",
		"replay_count", "is_attention_check", "created_at",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for rows.Next() {
		var (
			responseID       string
			participantID    string
			sessionToken     string
			studyID          string
			pairPresentation string
			sourceItemID     string
			taskOrder        int
			choice           string
			reasonCodes      string
			confidence       string
			responseTimeMS   string
			replayCount      int
			isAttentionCheck bool
			createdAt        time.Time
		)
		if err := rows.Scan(
			&responseID, &participantID, &sessionToken, &studyID,
			&pairPresentation, &sourceItemID, &taskOrder, &choice,
			&reasonCodes, &confidence, &responseTimeMS, &replayCount,
			&isAttentionCheck, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("export csv scan: %w", err)
		}
		rec := []string{
			responseID,
			participantID,
			sessionToken,
			studyID,
			pairPresentation,
			sourceItemID,
			strconv.Itoa(taskOrder),
			choice,
			reasonCodes,
			confidence,
			responseTimeMS,
			strconv.Itoa(replayCount),
			strconv.FormatBool(isAttentionCheck),
			createdAt.UTC().Format(time.RFC3339Nano),
		}
		if err := writer.Write(rec); err != nil {
			return nil, err
		}
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ExportService) ExportJSON(ctx context.Context) ([]byte, error) {
	csvBytes, err := s.ExportCSV(ctx)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bytes.NewReader(csvBytes))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []byte("[]"), nil
	}
	headers := records[0]
	out := make([]map[string]string, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		obj := make(map[string]string, len(headers))
		for j := range headers {
			obj[headers[j]] = records[i][j]
		}
		out = append(out, obj)
	}
	return json.Marshal(out)
}

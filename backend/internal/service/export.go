package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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
			pp.is_attention_check::text,
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
			isAttentionCheck string
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
			fmt.Sprintf("%d", replayCount),
			isAttentionCheck,
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

// ExportStudyCSV returns a CSV of all responses for the given study with
// computed columns (candidate_position, candidate_chosen, reason_*, is_suspect).
func (s *ExportService) ExportStudyCSV(ctx context.Context, studyID uuid.UUID) ([]byte, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			r.id,
			r.participant_id,
			st.name,
			g.name,
			si.pair_code,
			p.quality_flag,
			pp.left_method_type,
			pp.right_method_type,
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
		JOIN source_items si ON si.id = pp.source_item_id
		JOIN groups g ON g.id = si.group_id
		JOIN studies st ON st.id = p.study_id
		WHERE p.study_id = $1
		ORDER BY r.created_at ASC`, studyID)
	if err != nil {
		return nil, fmt.Errorf("export study csv query: %w", err)
	}
	defer rows.Close()

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	headers := []string{
		"response_id", "participant_id", "study_name", "group_name", "pair_code",
		"is_suspect", "candidate_position", "candidate_chosen",
		"reason_motion", "reason_artifacts", "reason_overall", "reason_integration",
		"confidence", "response_time_ms", "replay_count",
		"is_attention_check", "created_at",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for rows.Next() {
		var (
			responseID      string
			participantID   string
			studyName       string
			groupName       string
			pairCode        string
			qualityFlag     string
			leftMethodType  string
			rightMethodType string
			choice          string
			reasonCodes     string
			confidence      string
			responseTimeMS  string
			replayCount     int
			isAttentionCheck bool
			createdAt       time.Time
		)
		if err := rows.Scan(
			&responseID, &participantID, &studyName, &groupName, &pairCode,
			&qualityFlag, &leftMethodType, &rightMethodType, &choice,
			&reasonCodes, &confidence, &responseTimeMS, &replayCount,
			&isAttentionCheck, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("export study csv scan: %w", err)
		}

		// is_suspect: quality_flag IN ('suspect', 'flagged')
		isSuspect := qualityFlag == "suspect" || qualityFlag == "flagged"

		// candidate_position: where is candidate (left or right)
		candidatePosition := ""
		if leftMethodType == "candidate" {
			candidatePosition = "left"
		} else if rightMethodType == "candidate" {
			candidatePosition = "right"
		}

		// candidate_chosen: true if candidate side was chosen
		candidateChosen := candidatePosition != "" && choice == candidatePosition

		// reason_* flags
		codes := strings.Split(reasonCodes, "|")
		hasCode := func(target string) bool {
			for _, c := range codes {
				if c == target {
					return true
				}
			}
			return false
		}
		reasonMotion      := hasCode("motion")
		reasonArtifacts   := hasCode("artifacts")
		reasonOverall     := hasCode("overall")
		reasonIntegration := hasCode("integration")

		rec := []string{
			responseID,
			participantID,
			studyName,
			groupName,
			pairCode,
			strconv.FormatBool(isSuspect),
			candidatePosition,
			strconv.FormatBool(candidateChosen),
			strconv.FormatBool(reasonMotion),
			strconv.FormatBool(reasonArtifacts),
			strconv.FormatBool(reasonOverall),
			strconv.FormatBool(reasonIntegration),
			confidence,
			responseTimeMS,
			fmt.Sprintf("%d", replayCount),
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

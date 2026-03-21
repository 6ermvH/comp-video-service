package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// Shared scan callback for ExportCSV / ExportStudyCSV (16 columns, new format).
func scanExportRow(dest ...any) error {
	*(dest[0].(*string)) = "r1"
	*(dest[1].(*string)) = "p1"
	*(dest[2].(*string)) = "Study A"
	*(dest[3].(*string)) = "flooding"
	*(dest[4].(*string)) = "Group 1"
	*(dest[5].(*string)) = "P001"
	*(dest[6].(*string)) = "suspect"
	*(dest[7].(*string)) = "candidate"
	*(dest[8].(*string)) = "baseline"
	*(dest[9].(*string)) = "left"
	*(dest[10].(*string)) = "motion|artifacts"
	*(dest[11].(*string)) = "4"
	*(dest[12].(*string)) = "2000"
	*(dest[13].(*int)) = 2
	*(dest[14].(*bool)) = false
	*(dest[15].(*time.Time)) = time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	return nil
}

var scan15Args = []any{
	gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	gomock.Any(),
}

func TestExportServiceExportCSVSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockexportDB(ctrl)
	rows := NewMockexportRows(ctrl)
	db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(rows, nil)
	gomock.InOrder(
		rows.EXPECT().Next().Return(true),
		rows.EXPECT().Scan(scan15Args...).DoAndReturn(scanExportRow),
		rows.EXPECT().Next().Return(false),
	)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close()

	svc := &ExportService{db: db}
	csvBytes, err := svc.ExportCSV(context.Background())
	if err != nil {
		t.Fatalf("ExportCSV error: %v", err)
	}
	csvText := string(csvBytes)
	if !strings.Contains(csvText, "response_id,participant_id,study_name") {
		t.Fatalf("unexpected csv header: %s", csvText)
	}
	if !strings.Contains(csvText, "r1,p1,Study A") {
		t.Fatalf("unexpected csv row: %s", csvText)
	}
	// is_suspect should be true (quality_flag=suspect)
	if !strings.Contains(csvText, "true") {
		t.Fatalf("expected is_suspect=true in row: %s", csvText)
	}
}

func TestExportServiceExportCSVErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := NewMockexportDB(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(nil, errors.New("q"))

		svc := &ExportService{db: db}
		if _, err := svc.ExportCSV(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("scan", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := NewMockexportDB(ctrl)
		rows := NewMockexportRows(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(rows, nil)
		rows.EXPECT().Next().Return(true)
		rows.EXPECT().Scan(scan15Args...).Return(errors.New("scan"))
		rows.EXPECT().Close()

		svc := &ExportService{db: db}
		if _, err := svc.ExportCSV(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rows err", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db := NewMockexportDB(ctrl)
		rows := NewMockexportRows(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(rows, nil)
		rows.EXPECT().Next().Return(false)
		rows.EXPECT().Err().Return(errors.New("rows")).Times(2)
		rows.EXPECT().Close()

		svc := &ExportService{db: db}
		if _, err := svc.ExportCSV(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestExportStudyCSVSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyID := uuid.New()
	db := NewMockexportDB(ctrl)
	rows := NewMockexportRows(ctrl)
	db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(rows, nil)
	gomock.InOrder(
		rows.EXPECT().Next().Return(true),
		rows.EXPECT().Scan(scan15Args...).DoAndReturn(scanExportRow),
		rows.EXPECT().Next().Return(false),
	)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close()

	svc := &ExportService{db: db}
	csvBytes, err := svc.ExportStudyCSV(context.Background(), studyID)
	if err != nil {
		t.Fatalf("ExportStudyCSV error: %v", err)
	}
	csvText := string(csvBytes)
	if !strings.Contains(csvText, "response_id,participant_id,study_name,effect_type,group_name,pair_code") {
		t.Fatalf("unexpected csv header: %s", csvText)
	}
	// is_suspect: quality_flag=suspect → true
	if !strings.Contains(csvText, "true") {
		t.Fatalf("expected is_suspect=true: %s", csvText)
	}
	// candidate_position: left_method_type=candidate → left
	if !strings.Contains(csvText, "left") {
		t.Fatalf("expected candidate_position=left: %s", csvText)
	}
	if !strings.Contains(csvText, "r1") {
		t.Fatalf("expected response id in csv: %s", csvText)
	}
	// reason_motion and reason_artifacts should be true
	if strings.Count(csvText, "true") < 2 {
		t.Fatalf("expected multiple true values in csv: %s", csvText)
	}
}

func TestExportStudyCSVCandidatePositionRight(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyID := uuid.New()
	db := NewMockexportDB(ctrl)
	rows := NewMockexportRows(ctrl)
	db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(rows, nil)
	gomock.InOrder(
		rows.EXPECT().Next().Return(true),
		rows.EXPECT().Scan(scan15Args...).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*string)) = "resp-2"
			*(dest[1].(*string)) = "part-2"
			*(dest[2].(*string)) = "Study B"
			*(dest[3].(*string)) = "flooding"
			*(dest[4].(*string)) = "Group 2"
			*(dest[5].(*string)) = "P002"
			*(dest[6].(*string)) = "ok"
			*(dest[7].(*string)) = "baseline"
			*(dest[8].(*string)) = "candidate"
			*(dest[9].(*string)) = "right" // choice matches candidate_position=right
			*(dest[10].(*string)) = "overall"
			*(dest[11].(*string)) = ""
			*(dest[12].(*string)) = ""
			*(dest[13].(*int)) = 0
			*(dest[14].(*bool)) = true
			*(dest[15].(*time.Time)) = time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
			return nil
		}),
		rows.EXPECT().Next().Return(false),
	)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close()

	svc := &ExportService{db: db}
	csvBytes, err := svc.ExportStudyCSV(context.Background(), studyID)
	if err != nil {
		t.Fatalf("ExportStudyCSV error: %v", err)
	}
	csvText := string(csvBytes)
	if !strings.Contains(csvText, "right") {
		t.Fatalf("expected candidate_position=right: %s", csvText)
	}
	if !strings.Contains(csvText, "resp-2") {
		t.Fatalf("expected resp-2 in csv: %s", csvText)
	}
}

func TestExportStudyCSVErrors(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		db := NewMockexportDB(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(nil, errors.New("q"))

		svc := &ExportService{db: db}
		if _, err := svc.ExportStudyCSV(context.Background(), studyID); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("scan error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		db := NewMockexportDB(ctrl)
		rows := NewMockexportRows(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(rows, nil)
		rows.EXPECT().Next().Return(true)
		rows.EXPECT().Scan(scan15Args...).Return(errors.New("scan"))
		rows.EXPECT().Close()

		svc := &ExportService{db: db}
		if _, err := svc.ExportStudyCSV(context.Background(), studyID); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rows err", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		db := NewMockexportDB(ctrl)
		rows := NewMockexportRows(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(rows, nil)
		rows.EXPECT().Next().Return(false)
		rows.EXPECT().Err().Return(errors.New("rows")).Times(2)
		rows.EXPECT().Close()

		svc := &ExportService{db: db}
		if _, err := svc.ExportStudyCSV(context.Background(), studyID); err == nil {
			t.Fatal("expected error")
		}
	})
}

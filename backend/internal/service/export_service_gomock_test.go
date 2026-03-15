package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestExportServiceExportCSVSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockexportDB(ctrl)
	rows := NewMockexportRows(ctrl)
	db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(rows, nil)
	gomock.InOrder(
		rows.EXPECT().Next().Return(true),
		rows.EXPECT().Scan(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(),
		).DoAndReturn(func(dest ...any) error {
			vals := []string{"r1", "p1", "tok", "s1", "pp1", "si1", "0", "left", "speed|quality", "5", "1234", "1", "true", "2026-03-15T00:00:00Z"}
			for i := range vals {
				*(dest[i].(*string)) = vals[i]
			}
			return nil
		}),
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
	if !strings.Contains(csvText, "response_id,participant_id,session_token") {
		t.Fatalf("unexpected csv header: %s", csvText)
	}
	if !strings.Contains(csvText, "r1,p1,tok") {
		t.Fatalf("unexpected csv row: %s", csvText)
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
		rows.EXPECT().Scan(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(),
		).Return(errors.New("scan"))
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

func TestExportServiceExportJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockexportDB(ctrl)
	rows := NewMockexportRows(ctrl)
	db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(rows, nil)
	gomock.InOrder(
		rows.EXPECT().Next().Return(true),
		rows.EXPECT().Scan(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(),
		).DoAndReturn(func(dest ...any) error {
			vals := []string{"r2", "p2", "tok2", "s2", "pp2", "si2", "1", "right", "", "", "", "0", "false", "2026-03-15T00:00:00Z"}
			for i := range vals {
				*(dest[i].(*string)) = vals[i]
			}
			return nil
		}),
		rows.EXPECT().Next().Return(false),
	)
	rows.EXPECT().Err().Return(nil)
	rows.EXPECT().Close()

	svc := &ExportService{db: db}
	jsonBytes, err := svc.ExportJSON(context.Background())
	if err != nil {
		t.Fatalf("ExportJSON error: %v", err)
	}
	jsonText := string(jsonBytes)
	if !strings.Contains(jsonText, `"response_id":"r2"`) {
		t.Fatalf("unexpected json: %s", jsonText)
	}
}

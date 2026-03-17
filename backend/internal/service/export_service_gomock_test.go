package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
			*(dest[0].(*string)) = "r1"
			*(dest[1].(*string)) = "p1"
			*(dest[2].(*string)) = "tok"
			*(dest[3].(*string)) = "s1"
			*(dest[4].(*string)) = "pp1"
			*(dest[5].(*string)) = "si1"
			*(dest[6].(*int)) = 0
			*(dest[7].(*string)) = "left"
			*(dest[8].(*string)) = "speed|quality"
			*(dest[9].(*string)) = "5"
			*(dest[10].(*string)) = "1234"
			*(dest[11].(*int)) = 1
			*(dest[12].(*string)) = "true"
			*(dest[13].(*time.Time)) = time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
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
			*(dest[0].(*string)) = "r2"
			*(dest[1].(*string)) = "p2"
			*(dest[2].(*string)) = "tok2"
			*(dest[3].(*string)) = "s2"
			*(dest[4].(*string)) = "pp2"
			*(dest[5].(*string)) = "si2"
			*(dest[6].(*int)) = 1
			*(dest[7].(*string)) = "right"
			*(dest[8].(*string)) = ""
			*(dest[9].(*string)) = ""
			*(dest[10].(*string)) = ""
			*(dest[11].(*int)) = 0
			*(dest[12].(*string)) = "false"
			*(dest[13].(*time.Time)) = time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
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

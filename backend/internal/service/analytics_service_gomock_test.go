package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestAnalyticsServiceOverviewSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockanalyticsResponseRepository(ctrl)
	db := NewMockanalyticsDB(ctrl)
	participantsRow := NewMockanalyticsRow(ctrl)
	sourceItemsRow := NewMockanalyticsRow(ctrl)
	tieRow := NewMockanalyticsRow(ctrl)
	candidateWinsRow := NewMockanalyticsRow(ctrl)
	completedRow := NewMockanalyticsRow(ctrl)
	effectRows := NewMockanalyticsRows(ctrl)
	groupRows := NewMockanalyticsRows(ctrl)

	repo.EXPECT().CountTotal(gomock.Any()).Return(int64(10), nil)

	gomock.InOrder(
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(participantsRow),
		participantsRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 7
			return nil
		}),
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(sourceItemsRow),
		sourceItemsRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 12
			return nil
		}),
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(tieRow),
		tieRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 4
			return nil
		}),
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(candidateWinsRow),
		candidateWinsRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 6
			return nil
		}),
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(completedRow),
		completedRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 5
			return nil
		}),
	)

	db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(effectRows, nil)
	gomock.InOrder(
		effectRows.EXPECT().Next().Return(true),
		effectRows.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*string)) = "blur"
			*(dest[1].(*int64)) = 6
			*(dest[2].(*float64)) = 0.5
			*(dest[3].(*float64)) = 0.6
			return nil
		}),
		effectRows.EXPECT().Next().Return(false),
	)
	effectRows.EXPECT().Err().Return(nil)
	effectRows.EXPECT().Close()

	db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(groupRows, nil)
	gomock.InOrder(
		groupRows.EXPECT().Next().Return(true),
		groupRows.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(dest ...any) error {
			gid := uuid.New()
			tieRate := 0.25
			*(dest[0].(*uuid.UUID)) = gid
			*(dest[1].(*string)) = "g1"
			*(dest[2].(*int64)) = 8
			*(dest[3].(**float64)) = &tieRate
			return nil
		}),
		groupRows.EXPECT().Next().Return(false),
	)
	groupRows.EXPECT().Err().Return(nil)
	groupRows.EXPECT().Close()

	svc := &AnalyticsService{db: db, responseRepo: repo}
	out, err := svc.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview error: %v", err)
	}
	if out.TotalResponses != 10 || out.TotalParticipants != 7 {
		t.Fatalf("unexpected totals: %+v", out)
	}
	if out.TieRate != 0.4 || out.CandidateWinRate != 0.6 || out.CompletionRate != (5.0/7.0) {
		t.Fatalf("unexpected rates: tie=%v candidate=%v completion=%v", out.TieRate, out.CandidateWinRate, out.CompletionRate)
	}
	if out.TotalSourceItems != 12 {
		t.Fatalf("unexpected total source items: %d", out.TotalSourceItems)
	}
	if len(out.Effects) != 1 || len(out.Groups) != 1 {
		t.Fatalf("unexpected aggregates: effects=%d groups=%d", len(out.Effects), len(out.Groups))
	}
}

func TestAnalyticsServiceOverviewErrors(t *testing.T) {
	t.Run("count total", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		repo.EXPECT().CountTotal(gomock.Any()).Return(int64(0), errors.New("boom"))

		svc := &AnalyticsService{db: db, responseRepo: repo}
		if _, err := svc.Overview(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("participants scan", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		row := NewMockanalyticsRow(ctrl)

		repo.EXPECT().CountTotal(gomock.Any()).Return(int64(1), nil)
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(row)
		row.EXPECT().Scan(gomock.Any()).Return(errors.New("scan"))

		svc := &AnalyticsService{db: db, responseRepo: repo}
		if _, err := svc.Overview(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("effect query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		participantsRow := NewMockanalyticsRow(ctrl)
		sourceItemsRow := NewMockanalyticsRow(ctrl)
		tieRow := NewMockanalyticsRow(ctrl)
		candidateWinsRow := NewMockanalyticsRow(ctrl)
		completedRow := NewMockanalyticsRow(ctrl)

		repo.EXPECT().CountTotal(gomock.Any()).Return(int64(1), nil)
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(participantsRow)
		participantsRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 1
			return nil
		})
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(sourceItemsRow)
		sourceItemsRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 1
			return nil
		})
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(tieRow)
		tieRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 0
			return nil
		})
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(candidateWinsRow)
		candidateWinsRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 0
			return nil
		})
		db.EXPECT().QueryRow(gomock.Any(), gomock.Any()).Return(completedRow)
		completedRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
			*(dest[0].(*int64)) = 0
			return nil
		})
		db.EXPECT().Query(gomock.Any(), gomock.Any()).Return(nil, errors.New("query"))

		svc := &AnalyticsService{db: db, responseRepo: repo}
		if _, err := svc.Overview(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAnalyticsServicePairBreakdown(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		rows := NewMockanalyticsRows(ctrl)

		db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(rows, nil)
		gomock.InOrder(
			rows.EXPECT().Next().Return(true),
			rows.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(dest ...any) error {
				*(dest[0].(*uuid.UUID)) = uuid.New()
				pairCode := "p1"
				diff := "easy"
				*(dest[1].(**string)) = &pairCode
				*(dest[2].(**string)) = &diff
				*(dest[3].(*string)) = "g1"
				*(dest[4].(*int64)) = 10
				*(dest[5].(*int64)) = 6
				*(dest[6].(*int64)) = 3
				*(dest[7].(*int64)) = 1
				return nil
			}),
			rows.EXPECT().Next().Return(false),
		)
		rows.EXPECT().Err().Return(nil)
		rows.EXPECT().Close()

		svc := &AnalyticsService{db: db, responseRepo: repo}
		out, err := svc.PairBreakdown(context.Background(), studyID)
		if err != nil {
			t.Fatalf("PairBreakdown error: %v", err)
		}
		if len(out) != 1 || out[0].CandidateWinRate != 0.6 {
			t.Fatalf("unexpected output: %+v", out)
		}
	})

	t.Run("query error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		db.EXPECT().Query(gomock.Any(), gomock.Any(), studyID).Return(nil, errors.New("q"))

		svc := &AnalyticsService{db: db, responseRepo: repo}
		if _, err := svc.PairBreakdown(context.Background(), studyID); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAnalyticsServiceStudyDetail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		rows := NewMockanalyticsRows(ctrl)

		repo.EXPECT().CountChoicesByStudy(gomock.Any(), studyID).Return(int64(3), int64(2), int64(1), nil)
		db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(rows, nil)
		gomock.InOrder(
			rows.EXPECT().Next().Return(true),
			rows.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(dest ...any) error {
				gid := uuid.New()
				*(dest[0].(*uuid.UUID)) = gid
				*(dest[1].(*string)) = "group"
				*(dest[2].(*int64)) = 5
				*(dest[3].(**float64)) = nil
				return nil
			}),
			rows.EXPECT().Next().Return(false),
		)
		rows.EXPECT().Err().Return(nil)
		rows.EXPECT().Close()

		svc := &AnalyticsService{db: db, responseRepo: repo}
		out, err := svc.StudyDetail(context.Background(), studyID)
		if err != nil {
			t.Fatalf("StudyDetail error: %v", err)
		}
		if out.Total != 6 || out.LeftWinRate != 0.5 || out.RightWinRate != (2.0/6.0) || out.TieRate != (1.0/6.0) {
			t.Fatalf("unexpected rates: %+v", out)
		}
		if len(out.Groups) != 1 || out.Groups[0].TieRate != 0 {
			t.Fatalf("unexpected groups: %+v", out.Groups)
		}
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		repo.EXPECT().CountChoicesByStudy(gomock.Any(), studyID).Return(int64(0), int64(0), int64(0), errors.New("boom"))

		svc := &AnalyticsService{db: db, responseRepo: repo}
		if _, err := svc.StudyDetail(context.Background(), studyID); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("group query error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		studyID := uuid.New()
		repo := NewMockanalyticsResponseRepository(ctrl)
		db := NewMockanalyticsDB(ctrl)
		repo.EXPECT().CountChoicesByStudy(gomock.Any(), studyID).Return(int64(0), int64(0), int64(0), nil)
		db.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("q"))

		svc := &AnalyticsService{db: db, responseRepo: repo}
		if _, err := svc.StudyDetail(context.Background(), studyID); err == nil {
			t.Fatal("expected error")
		}
	})
}

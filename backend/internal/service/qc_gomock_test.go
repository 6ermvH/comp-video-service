package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestQCServiceBuildReportWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resp := NewMockqcResponseRepository(ctrl)
	part := NewMockqcParticipantRepository(ctrl)
	svc := NewQCService(resp, part)

	resp.EXPECT().CountTotal(gomock.Any()).Return(int64(100), nil)
	resp.EXPECT().CountFastResponses(gomock.Any(), 1500).Return(int64(15), nil)
	resp.EXPECT().StraightLiningParticipants(gomock.Any()).Return(int64(2), nil)

	report, err := svc.BuildReport(context.Background())
	if err != nil {
		t.Fatalf("BuildReport error: %v", err)
	}
	if report.TotalResponses != 100 || report.FastResponses != 15 || report.StraightLiningProfiles != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestQCServiceEvaluateParticipantFinalFlagBranches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	participantID := uuid.New()

	resp := NewMockqcResponseRepository(ctrl)
	part := NewMockqcParticipantRepository(ctrl)
	svc := NewQCService(resp, part)

	resp.EXPECT().CountByParticipant(gomock.Any(), participantID).Return(int64(10), nil)
	resp.EXPECT().CountFastByParticipant(gomock.Any(), participantID, 1500).Return(int64(2), nil)
	resp.EXPECT().AttentionCheckStats(gomock.Any(), participantID).Return(int64(1), int64(1), nil)
	part.EXPECT().UpdateQualityFlag(gomock.Any(), participantID, "flagged").Return(nil)

	if err := svc.EvaluateParticipantFinalFlag(context.Background(), participantID); err != nil {
		t.Fatalf("EvaluateParticipantFinalFlag error: %v", err)
	}
}

func TestQCServiceUpdateParticipantFlag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	participantID := uuid.New()
	resp := NewMockqcResponseRepository(ctrl)
	part := NewMockqcParticipantRepository(ctrl)
	svc := NewQCService(resp, part)

	ms := 300
	part.EXPECT().UpdateQualityFlag(gomock.Any(), participantID, "suspect").Return(nil)
	if err := svc.UpdateParticipantFlag(context.Background(), participantID, &ms); err != nil {
		t.Fatalf("UpdateParticipantFlag error: %v", err)
	}
}

func TestQCServiceEvaluateParticipantFinalFlagSuspectAndOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	participantID := uuid.New()

	resp := NewMockqcResponseRepository(ctrl)
	part := NewMockqcParticipantRepository(ctrl)
	svc := NewQCService(resp, part)

	// suspect branch
	resp.EXPECT().CountByParticipant(gomock.Any(), participantID).Return(int64(10), nil)
	resp.EXPECT().CountFastByParticipant(gomock.Any(), participantID, 1500).Return(int64(4), nil)
	resp.EXPECT().AttentionCheckStats(gomock.Any(), participantID).Return(int64(0), int64(0), nil)
	part.EXPECT().UpdateQualityFlag(gomock.Any(), participantID, "suspect").Return(nil)
	if err := svc.EvaluateParticipantFinalFlag(context.Background(), participantID); err != nil {
		t.Fatalf("suspect branch err: %v", err)
	}

	// ok branch
	resp.EXPECT().CountByParticipant(gomock.Any(), participantID).Return(int64(10), nil)
	resp.EXPECT().CountFastByParticipant(gomock.Any(), participantID, 1500).Return(int64(1), nil)
	resp.EXPECT().AttentionCheckStats(gomock.Any(), participantID).Return(int64(0), int64(0), nil)
	part.EXPECT().UpdateQualityFlag(gomock.Any(), participantID, "ok").Return(nil)
	if err := svc.EvaluateParticipantFinalFlag(context.Background(), participantID); err != nil {
		t.Fatalf("ok branch err: %v", err)
	}
}

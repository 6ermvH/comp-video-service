package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"comp-video-service/backend/internal/model"
)

func TestAssignmentServiceNoItems(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sourceRepo := NewMockassignmentSourceItemRepository(ctrl)
	groupRepo := NewMockassignmentGroupRepository(ctrl)
	videoRepo := NewMockassignmentVideoRepository(ctrl)
	pairRepo := NewMockassignmentPairRepository(ctrl)
	svc := NewAssignmentService(sourceRepo, groupRepo, videoRepo, pairRepo)

	sid := uuid.New()
	sourceRepo.EXPECT().ListByStudy(gomock.Any(), sid).Return([]*model.SourceItem{}, nil)

	created, err := svc.AssignForParticipant(context.Background(), uuid.New(), sid, 10)
	if err != nil {
		t.Fatalf("AssignForParticipant err: %v", err)
	}
	if created != 0 {
		t.Fatalf("expected 0 created, got %d", created)
	}
}

func TestAssignmentServiceCreatesTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sourceRepo := NewMockassignmentSourceItemRepository(ctrl)
	groupRepo := NewMockassignmentGroupRepository(ctrl)
	videoRepo := NewMockassignmentVideoRepository(ctrl)
	pairRepo := NewMockassignmentPairRepository(ctrl)
	svc := NewAssignmentService(sourceRepo, groupRepo, videoRepo, pairRepo)

	studyID := uuid.New()
	groupID := uuid.New()
	itemID := uuid.New()
	pID := uuid.New()

	sourceRepo.EXPECT().ListByStudy(gomock.Any(), studyID).Return([]*model.SourceItem{{ID: itemID, GroupID: groupID}}, nil)
	groupRepo.EXPECT().ListByStudy(gomock.Any(), studyID).Return([]*model.Group{{ID: groupID, TargetVotesPerPair: 10}}, nil)
	sourceRepo.EXPECT().ResponseCountsByStudy(gomock.Any(), studyID).Return(map[uuid.UUID]int64{itemID: 0}, nil)
	videoRepo.EXPECT().ListBySourceItem(gomock.Any(), itemID).Return([]*model.Video{{ID: uuid.New()}, {ID: uuid.New()}}, nil)
	pairRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.PairPresentation{ID: uuid.New()}, nil)

	created, err := svc.AssignForParticipant(context.Background(), pID, studyID, 1)
	if err != nil {
		t.Fatalf("AssignForParticipant err: %v", err)
	}
	if created != 1 {
		t.Fatalf("expected 1 created, got %d", created)
	}
}

func TestDerefOrEmpty(t *testing.T) {
	if v := derefOrEmpty(nil); v != "" {
		t.Fatalf("expected empty, got %q", v)
	}
	s := "x"
	if v := derefOrEmpty(&s); v != "x" {
		t.Fatalf("expected x, got %q", v)
	}
}

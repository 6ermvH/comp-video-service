package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"comp-video-service/backend/internal/model"
)

func TestStudyServiceBasicMethodsWithGomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMockstudyRepository(ctrl)
	groupRepo := NewMockgroupRepository(ctrl)
	sourceRepo := NewMocksourceItemRepository(ctrl)
	videoRepo := NewMockstudyVideoRepository(ctrl)
	svc := NewStudyService(studyRepo, groupRepo, sourceRepo, videoRepo)

	studyRepo.EXPECT().List(gomock.Any()).Return([]*model.Study{{ID: uuid.New()}}, nil)
	if _, err := svc.ListStudies(context.Background()); err != nil {
		t.Fatalf("ListStudies: %v", err)
	}

	studyRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.Study{ID: uuid.New()}, nil)
	if _, err := svc.CreateStudy(context.Background(), &model.CreateStudyRequest{Name: "s", EffectType: "flooding"}); err != nil {
		t.Fatalf("CreateStudy: %v", err)
	}

	gid := uuid.New()
	groupRepo.EXPECT().Create(gomock.Any(), gid, gomock.Any()).Return(&model.Group{ID: uuid.New()}, nil)
	if _, err := svc.CreateGroup(context.Background(), gid, &model.CreateGroupRequest{Name: "g"}); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	groupRepo.EXPECT().ListByStudy(gomock.Any(), gid).Return([]*model.Group{}, nil)
	if _, err := svc.ListGroups(context.Background(), gid); err != nil {
		t.Fatalf("ListGroups: %v", err)
	}

	sourceRepo.EXPECT().ListWithFilters(gomock.Any(), gomock.Nil(), gomock.Nil()).Return([]*model.SourceItem{}, nil)
	if _, err := svc.ListSourceItems(context.Background(), nil, nil); err != nil {
		t.Fatalf("ListSourceItems: %v", err)
	}

	videoRepo.EXPECT().ListAll(gomock.Any()).Return([]*model.Video{}, nil)
	if _, err := svc.ListAssets(context.Background()); err != nil {
		t.Fatalf("ListAssets: %v", err)
	}

	studyID := uuid.New()
	groupID2 := uuid.New()
	baselineID := uuid.New()
	candidateID := uuid.New()
	createdItem := &model.SourceItem{ID: uuid.New(), StudyID: studyID, GroupID: groupID2}
	sourceRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(createdItem, nil)
	videoRepo.EXPECT().Link(gomock.Any(), baselineID, createdItem.ID, "baseline").Return(nil)
	videoRepo.EXPECT().Link(gomock.Any(), candidateID, createdItem.ID, "candidate").Return(nil)
	if _, err := svc.CreatePair(context.Background(), studyID, &model.CreatePairRequest{
		GroupID:          groupID2,
		BaselineVideoID:  baselineID,
		CandidateVideoID: candidateID,
	}); err != nil {
		t.Fatalf("CreatePair: %v", err)
	}
}

func TestStudyServiceUpdateStatusValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMockstudyRepository(ctrl)
	svc := NewStudyService(studyRepo, NewMockgroupRepository(ctrl), NewMocksourceItemRepository(ctrl), nil)

	if _, err := svc.UpdateStatus(context.Background(), uuid.New(), "bad"); err == nil {
		t.Fatal("expected invalid status error")
	}

	id := uuid.New()
	studyRepo.EXPECT().UpdateStatus(gomock.Any(), id, "active").Return(&model.Study{ID: id, Status: "active"}, nil)
	if _, err := svc.UpdateStatus(context.Background(), id, "ACTIVE"); err != nil {
		t.Fatalf("UpdateStatus active: %v", err)
	}
}

func TestStudyServiceUpdateStudyValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMockstudyRepository(ctrl)
	svc := NewStudyService(studyRepo, NewMockgroupRepository(ctrl), NewMocksourceItemRepository(ctrl), nil)
	id := uuid.New()

	badStatus := "bad"
	if _, err := svc.UpdateStudy(context.Background(), id, &model.UpdateStudyRequest{Status: &badStatus}); err == nil {
		t.Fatal("expected invalid status error")
	}

	name := "New name"
	status := "ACTIVE"
	studyRepo.EXPECT().Update(gomock.Any(), id, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error) {
			if req.Status == nil || *req.Status != "active" {
				t.Fatalf("expected normalized status active, got %+v", req.Status)
			}
			if req.Name == nil || *req.Name != name {
				t.Fatalf("expected name to pass through")
			}
			return &model.Study{ID: id, Status: "active", Name: name}, nil
		},
	)
	if _, err := svc.UpdateStudy(context.Background(), id, &model.UpdateStudyRequest{
		Status: &status,
		Name:   &name,
	}); err != nil {
		t.Fatalf("UpdateStudy: %v", err)
	}
}

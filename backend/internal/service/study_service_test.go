package service

import (
	"context"
	"strings"
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
	svc := NewStudyService(studyRepo, groupRepo, sourceRepo, nil)

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

func TestStudyServiceImportSourceItemsCSV(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sourceRepo := NewMocksourceItemRepository(ctrl)
	svc := NewStudyService(NewMockstudyRepository(ctrl), NewMockgroupRepository(ctrl), sourceRepo, nil)

	studyID := uuid.New()
	groupID := uuid.New()
	csvData := "group_id,source_image_id,pair_code,difficulty,is_attention_check,notes\n" +
		groupID.String() + ",img,pair,hard,true,note\n"

	sourceRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, si *model.SourceItem) (*model.SourceItem, error) {
		if si.StudyID != studyID || si.GroupID != groupID {
			t.Fatalf("unexpected source item payload: %+v", si)
		}
		return &model.SourceItem{ID: uuid.New()}, nil
	})

	created, err := svc.ImportSourceItemsCSV(context.Background(), studyID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ImportSourceItemsCSV: %v", err)
	}
	if created != 1 {
		t.Fatalf("expected created=1 got %d", created)
	}
}

func TestStudyServiceImportSourceItemsCSV_LinksExistingVideos(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sourceRepo := NewMocksourceItemRepository(ctrl)
	videoRepo := NewMockstudyVideoRepository(ctrl)
	svc := NewStudyService(NewMockstudyRepository(ctrl), NewMockgroupRepository(ctrl), sourceRepo, videoRepo)

	studyID := uuid.New()
	groupID := uuid.New()
	sourceItemID := uuid.New()
	csvData := "group_id,source_image_id,pair_code,difficulty,is_attention_check,notes,baseline_s3_key,candidate_s3_key\n" +
		groupID.String() + ",img,pair,hard,false,note,s3://baseline.mp4,s3://candidate.mp4\n"

	sourceRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.SourceItem{ID: sourceItemID}, nil)
	videoRepo.EXPECT().LinkOrCreate(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
		func(_ context.Context, v *model.Video) (*model.Video, error) {
			if v.SourceItemID == nil || *v.SourceItemID != sourceItemID {
				t.Fatalf("source item id not linked: %+v", v.SourceItemID)
			}
			if v.MethodType == nil {
				t.Fatalf("method type is nil")
			}
			return v, nil
		},
	)

	created, err := svc.ImportSourceItemsCSV(context.Background(), studyID, strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ImportSourceItemsCSV: %v", err)
	}
	if created != 1 {
		t.Fatalf("expected created=1 got %d", created)
	}
}

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
)

// StudyService provides admin operations around studies/groups/source items.
type StudyService struct {
	studyRepo      studyRepository
	groupRepo      groupRepository
	sourceItemRepo sourceItemRepository
	videoRepo      studyVideoRepository
}

type studyRepository interface {
	List(ctx context.Context) ([]*model.Study, error)
	Create(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Study, error)
	Update(ctx context.Context, id uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error)
}

type groupRepository interface {
	Create(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error)
	ListByStudy(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error)
}

type sourceItemRepository interface {
	Create(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error)
	ListWithFilters(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItem, error)
	ListWithDetails(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItemDetail, error)
	Delete(ctx context.Context, id uuid.UUID) (bool, error)
	UpdateAttentionCheck(ctx context.Context, id uuid.UUID, isAttentionCheck bool) error
	GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*model.SourceItemDetail, error)
}

type studyVideoRepository interface {
	LinkOrCreate(ctx context.Context, v *model.Video) (*model.Video, error)
	Link(ctx context.Context, videoID uuid.UUID, sourceItemID uuid.UUID, methodType string) error
	ListAll(ctx context.Context) ([]*model.Video, error)
	ListPaged(ctx context.Context, page, perPage int, search string) ([]*model.Video, int, error)
	ListFree(ctx context.Context) ([]*model.Video, error)
}

var ErrPairHasResponses = errors.New("pair has participant responses and cannot be deleted")

//go:generate go run go.uber.org/mock/mockgen -source=study.go -destination=study_mocks_test.go -package=service

func NewStudyService(
	studyRepo studyRepository,
	groupRepo groupRepository,
	sourceItemRepo sourceItemRepository,
	videoRepo studyVideoRepository,
) *StudyService {
	return &StudyService{
		studyRepo:      studyRepo,
		groupRepo:      groupRepo,
		sourceItemRepo: sourceItemRepo,
		videoRepo:      videoRepo,
	}
}

func (s *StudyService) ListStudies(ctx context.Context) ([]*model.Study, error) {
	return s.studyRepo.List(ctx)
}

func (s *StudyService) CreateStudy(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
	return s.studyRepo.Create(ctx, req)
}

func (s *StudyService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Study, error) {
	normalized := strings.ToLower(status)
	switch normalized {
	case "draft", "active", "paused", "archived":
	default:
		return nil, fmt.Errorf("unsupported status: %s", status)
	}
	return s.studyRepo.UpdateStatus(ctx, id, normalized)
}

func (s *StudyService) UpdateStudy(ctx context.Context, id uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error) {
	if req.Status != nil {
		normalized := strings.ToLower(*req.Status)
		switch normalized {
		case "draft", "active", "paused", "archived":
			req.Status = &normalized
		default:
			return nil, fmt.Errorf("unsupported status: %s", *req.Status)
		}
	}
	return s.studyRepo.Update(ctx, id, req)
}

func (s *StudyService) CreateGroup(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
	return s.groupRepo.Create(ctx, studyID, req)
}

func (s *StudyService) ListGroups(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error) {
	return s.groupRepo.ListByStudy(ctx, studyID)
}

func (s *StudyService) ListSourceItems(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItemDetail, error) {
	return s.sourceItemRepo.ListWithDetails(ctx, studyID, groupID)
}

func (s *StudyService) ListAssets(ctx context.Context, page, perPage int, search string) ([]*model.Video, int, error) {
	return s.videoRepo.ListPaged(ctx, page, perPage, search)
}

func (s *StudyService) ListFreeAssets(ctx context.Context) ([]*model.Video, error) {
	return s.videoRepo.ListFree(ctx)
}

func (s *StudyService) CreatePair(ctx context.Context, studyID uuid.UUID, req *model.CreatePairRequest) (*model.SourceItem, error) {
	item, err := s.sourceItemRepo.Create(ctx, &model.SourceItem{
		StudyID:          studyID,
		GroupID:          req.GroupID,
		PairCode:         nilIfEmpty(req.PairCode),
		Difficulty:       nilIfEmpty(req.Difficulty),
		Notes:            nilIfEmpty(req.Notes),
		IsAttentionCheck: req.IsAttentionCheck,
	})
	if err != nil {
		return nil, fmt.Errorf("create source item: %w", err)
	}
	if err := s.videoRepo.Link(ctx, req.BaselineVideoID, item.ID, "baseline"); err != nil {
		return nil, fmt.Errorf("link baseline: %w", err)
	}
	if err := s.videoRepo.Link(ctx, req.CandidateVideoID, item.ID, "candidate"); err != nil {
		return nil, fmt.Errorf("link candidate: %w", err)
	}
	return item, nil
}

func (s *StudyService) UpdateSourceItemAttention(ctx context.Context, id uuid.UUID, isAttentionCheck bool) (*model.SourceItemDetail, error) {
	if err := s.sourceItemRepo.UpdateAttentionCheck(ctx, id, isAttentionCheck); err != nil {
		return nil, fmt.Errorf("update attention check: %w", err)
	}
	return s.sourceItemRepo.GetByIDWithDetails(ctx, id)
}

func (s *StudyService) DeletePair(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.sourceItemRepo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrPairHasResponses
	}
	return nil
}

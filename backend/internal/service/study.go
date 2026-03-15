package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
)

// StudyService provides admin operations around studies/groups/source items.
type StudyService struct {
	studyRepo      *repository.StudyRepository
	groupRepo      *repository.GroupRepository
	sourceItemRepo *repository.SourceItemRepository
	videoRepo      *repository.VideoRepository
}

func NewStudyService(
	studyRepo *repository.StudyRepository,
	groupRepo *repository.GroupRepository,
	sourceItemRepo *repository.SourceItemRepository,
	videoRepo *repository.VideoRepository,
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

func (s *StudyService) CreateGroup(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
	return s.groupRepo.Create(ctx, studyID, req)
}

func (s *StudyService) ListGroups(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error) {
	return s.groupRepo.ListByStudy(ctx, studyID)
}

func (s *StudyService) ListSourceItems(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItem, error) {
	return s.sourceItemRepo.ListWithFilters(ctx, studyID, groupID)
}

// ImportSourceItemsCSV imports source-items with optional asset keys.
// Columns:
// group_id,source_image_id,pair_code,difficulty,is_attention_check,notes,baseline_s3_key,candidate_s3_key
func (s *StudyService) ImportSourceItemsCSV(ctx context.Context, studyID uuid.UUID, r io.Reader) (int, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("read csv: %w", err)
	}
	if len(records) <= 1 {
		return 0, nil
	}

	created := 0
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 6 {
			continue
		}
		groupID, err := uuid.Parse(strings.TrimSpace(row[0]))
		if err != nil {
			continue
		}
		attention := strings.EqualFold(strings.TrimSpace(row[4]), "true") || strings.TrimSpace(row[4]) == "1"
		item, err := s.sourceItemRepo.Create(ctx, &model.SourceItem{
			StudyID:          studyID,
			GroupID:          groupID,
			SourceImageID:    nilIfEmpty(strings.TrimSpace(row[1])),
			PairCode:         nilIfEmpty(strings.TrimSpace(row[2])),
			Difficulty:       nilIfEmpty(strings.TrimSpace(row[3])),
			IsAttentionCheck: attention,
			Notes:            nilIfEmpty(strings.TrimSpace(row[5])),
		})
		if err != nil {
			continue
		}

		// Optional columns with pre-uploaded object keys.
		var baselineKey, candidateKey string
		if len(row) > 6 {
			baselineKey = strings.TrimSpace(row[6])
		}
		if len(row) > 7 {
			candidateKey = strings.TrimSpace(row[7])
		}
		if baselineKey != "" {
			sourceItemID := item.ID
			_, _ = s.videoRepo.Create(ctx, nil, &model.Video{
				SourceItemID: &sourceItemID,
				MethodType:   nilIfEmpty("baseline"),
				Title:        strings.TrimSpace(row[2]) + " baseline",
				Description:  nilIfEmpty("imported from CSV"),
				S3Key:        baselineKey,
				Status:       model.VideoStatusActive,
			})
		}
		if candidateKey != "" {
			sourceItemID := item.ID
			_, _ = s.videoRepo.Create(ctx, nil, &model.Video{
				SourceItemID: &sourceItemID,
				MethodType:   nilIfEmpty("candidate"),
				Title:        strings.TrimSpace(row[2]) + " candidate",
				Description:  nilIfEmpty("imported from CSV"),
				S3Key:        candidateKey,
				Status:       model.VideoStatusActive,
			})
		}
		created++
	}
	return created, nil
}

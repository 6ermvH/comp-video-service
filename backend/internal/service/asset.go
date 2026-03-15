package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/storage"
)

// AssetUploadInput describes one uploaded video asset.
type AssetUploadInput struct {
	SourceItemID *uuid.UUID
	MethodType   string
	Title        string
	Description  string
	ContentType  string
	Filename     string
	Size         int64
	Reader       io.Reader
}

// AssetService handles video_assets upload and persistence.
type AssetService struct {
	videoRepo *repository.VideoRepository
	s3        *storage.S3Client
}

func NewAssetService(videoRepo *repository.VideoRepository, s3 *storage.S3Client) *AssetService {
	return &AssetService{videoRepo: videoRepo, s3: s3}
}

func (s *AssetService) Upload(ctx context.Context, input AssetUploadInput) (*model.Video, error) {
	method := strings.ToLower(strings.TrimSpace(input.MethodType))
	if method != "baseline" && method != "candidate" {
		return nil, fmt.Errorf("method_type must be baseline or candidate")
	}

	key := fmt.Sprintf("videos/%s.mp4", uuid.NewString())
	if err := s.s3.Upload(ctx, key, input.ContentType, input.Reader, input.Size); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = strings.TrimSuffix(input.Filename, ".mp4")
	}

	asset, err := s.videoRepo.Create(ctx, nil, &model.Video{
		SourceItemID: input.SourceItemID,
		MethodType:   nilIfEmpty(method),
		Title:        title,
		Description:  nilIfEmpty(input.Description),
		S3Key:        key,
		Status:       model.VideoStatusActive,
	})
	if err != nil {
		_ = s.s3.Delete(ctx, key)
		return nil, err
	}
	return asset, nil
}

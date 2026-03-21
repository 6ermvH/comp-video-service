package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
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
	videoRepo assetVideoRepository
	s3        assetStorage
}

type assetVideoRepository interface {
	Create(ctx context.Context, m *model.Video) (*model.Video, error)
	Delete(ctx context.Context, id uuid.UUID) (bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Video, error)
}

type assetStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) error
	Delete(ctx context.Context, key string) error
	PresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

//go:generate go run go.uber.org/mock/mockgen -source=asset.go -destination=asset_mocks_test.go -package=service

type videoRepositoryAdapter struct {
	repo *repository.VideoRepository
}

func (a videoRepositoryAdapter) Create(ctx context.Context, m *model.Video) (*model.Video, error) {
	return a.repo.Create(ctx, nil, m)
}

func (a videoRepositoryAdapter) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	return a.repo.Delete(ctx, id)
}

func (a videoRepositoryAdapter) GetByID(ctx context.Context, id uuid.UUID) (*model.Video, error) {
	return a.repo.GetByID(ctx, id)
}

var (
	ErrAssetInUse   = errors.New("video asset is linked to a pair or referenced in presentations")
	ErrAssetNotFound = errors.New("video asset not found")
)

func NewAssetService(videoRepo *repository.VideoRepository, s3 assetStorage) *AssetService {
	return newAssetServiceWithDeps(videoRepositoryAdapter{repo: videoRepo}, s3)
}

func newAssetServiceWithDeps(videoRepo assetVideoRepository, s3 assetStorage) *AssetService {
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

	asset, err := s.videoRepo.Create(ctx, &model.Video{
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

func (s *AssetService) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.videoRepo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrAssetInUse
	}
	return nil
}

// GetPresignedURL returns a presigned (or public) URL for the given asset.
func (s *AssetService) GetPresignedURL(ctx context.Context, id uuid.UUID) (string, error) {
	video, err := s.videoRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get asset: %w", ErrAssetNotFound)
	}
	url, err := s.s3.PresignedURL(ctx, video.S3Key, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	return url, nil
}

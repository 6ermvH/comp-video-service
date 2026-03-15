package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"comp-video-service/backend/internal/model"
	"go.uber.org/mock/gomock"
)

func TestAssetServiceUpload(t *testing.T) {
	t.Run("invalid method", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockassetVideoRepository(ctrl)
		s3 := NewMockassetStorage(ctrl)
		svc := newAssetServiceWithDeps(repo, s3)

		_, err := svc.Upload(context.Background(), AssetUploadInput{MethodType: "wrong"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("upload error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockassetVideoRepository(ctrl)
		s3 := NewMockassetStorage(ctrl)
		s3.EXPECT().Upload(gomock.Any(), gomock.Any(), "video/mp4", gomock.Any(), int64(10)).Return(errors.New("upload"))

		svc := newAssetServiceWithDeps(repo, s3)
		_, err := svc.Upload(context.Background(), AssetUploadInput{
			MethodType:  "baseline",
			ContentType: "video/mp4",
			Filename:    "clip.mp4",
			Size:        10,
			Reader:      bytes.NewReader([]byte("1234567890")),
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("repo error triggers delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockassetVideoRepository(ctrl)
		s3 := NewMockassetStorage(ctrl)
		s3.EXPECT().Upload(gomock.Any(), gomock.Any(), "video/mp4", gomock.Any(), int64(4)).Return(nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, errors.New("db"))
		s3.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)

		svc := newAssetServiceWithDeps(repo, s3)
		_, err := svc.Upload(context.Background(), AssetUploadInput{
			MethodType:  "candidate",
			ContentType: "video/mp4",
			Filename:    "sample.mp4",
			Size:        4,
			Reader:      bytes.NewReader([]byte("data")),
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("success uses filename as title", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := NewMockassetVideoRepository(ctrl)
		s3 := NewMockassetStorage(ctrl)
		s3.EXPECT().Upload(gomock.Any(), gomock.Any(), "video/mp4", gomock.Any(), int64(4)).Return(nil)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, v *model.Video) (*model.Video, error) {
			if v.Title != "render" {
				t.Fatalf("unexpected title: %q", v.Title)
			}
			if v.MethodType == nil || *v.MethodType != "baseline" {
				t.Fatalf("unexpected method type: %+v", v.MethodType)
			}
			if v.Status != model.VideoStatusActive {
				t.Fatalf("unexpected status: %s", v.Status)
			}
			return v, nil
		})

		svc := newAssetServiceWithDeps(repo, s3)
		out, err := svc.Upload(context.Background(), AssetUploadInput{
			MethodType:  " baseline ",
			ContentType: "video/mp4",
			Filename:    "render.mp4",
			Size:        4,
			Reader:      bytes.NewReader([]byte("data")),
		})
		if err != nil {
			t.Fatalf("Upload error: %v", err)
		}
		if out == nil {
			t.Fatal("expected video")
		}
	})
}

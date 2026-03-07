package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

func TestDownloadVideoUseCase_Execute_Success(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{presignURL: "http://minio/frames.zip"}
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	v.MarkCompleted("zips/frames.zip", 30, 2.5)
	repo.videos[v.ID] = v

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	result, err := uc.Execute(context.Background(), v.ID, "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.URL != "http://minio/frames.zip" {
		t.Errorf("expected URL=http://minio/frames.zip, got %s", result.URL)
	}
	if result.ZipKey != "zips/frames.zip" {
		t.Errorf("expected ZipKey=zips/frames.zip, got %s", result.ZipKey)
	}
	if result.ExpiresSeconds <= 0 {
		t.Error("expected positive ExpiresSeconds")
	}
}

func TestDownloadVideoUseCase_Execute_NotFound(t *testing.T) {
	repo := newMockRepo()
	repo.findErr = errors.New("not found")
	storage := &mockStorage{}
	logger := zap.NewNop()

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	_, err := uc.Execute(context.Background(), [16]byte{1}, "user-1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestDownloadVideoUseCase_Execute_WrongUser(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	v.MarkCompleted("zips/frames.zip", 30, 2.5)
	repo.videos[v.ID] = v

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	_, err := uc.Execute(context.Background(), v.ID, "user-attacker")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestDownloadVideoUseCase_Execute_VideoNotReady_Pending(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	// Status is PENDING by default
	repo.videos[v.ID] = v

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	_, err := uc.Execute(context.Background(), v.ID, "user-1")

	if !errors.Is(err, usecase.ErrVideoNotReady) {
		t.Errorf("expected ErrVideoNotReady, got %v", err)
	}
}

func TestDownloadVideoUseCase_Execute_VideoNotReady_Processing(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	v.MarkProcessing()
	repo.videos[v.ID] = v

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	_, err := uc.Execute(context.Background(), v.ID, "user-1")

	if !errors.Is(err, usecase.ErrVideoNotReady) {
		t.Errorf("expected ErrVideoNotReady, got %v", err)
	}
}

func TestDownloadVideoUseCase_Execute_VideoFailed(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	v.MarkFailed("ffmpeg error")
	repo.videos[v.ID] = v

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	_, err := uc.Execute(context.Background(), v.ID, "user-1")

	if !errors.Is(err, usecase.ErrVideoFailed) {
		t.Errorf("expected ErrVideoFailed, got %v", err)
	}
}

func TestDownloadVideoUseCase_Execute_PresignError(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{presignErr: errors.New("presign failed")}
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	v.MarkCompleted("zips/frames.zip", 30, 2.5)
	repo.videos[v.ID] = v

	uc := usecase.NewDownloadVideoUseCase(repo, storage, logger)
	_, err := uc.Execute(context.Background(), v.ID, "user-1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

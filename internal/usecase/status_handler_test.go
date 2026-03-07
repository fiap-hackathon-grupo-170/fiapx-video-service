package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

func TestHandleStatusUpdateUseCase_Execute_Completed(t *testing.T) {
	repo := newMockRepo()
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewHandleStatusUpdateUseCase(repo, cache, logger)
	err := uc.Execute(context.Background(), entity.VideoStatusMessage{
		JobID:      v.ID,
		UserID:     "user-1",
		Status:     "COMPLETED",
		ZipKey:     "zips/frames.zip",
		FrameCount: 30,
		Duration:   2.5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	updated := repo.videos[v.ID]
	if updated.Status != entity.VideoStatusCompleted {
		t.Errorf("expected COMPLETED, got %s", updated.Status)
	}
	if updated.ZipKey != "zips/frames.zip" {
		t.Errorf("expected ZipKey=zips/frames.zip, got %s", updated.ZipKey)
	}
	if len(cache.invalidateCalls) != 1 {
		t.Errorf("expected 1 cache invalidation, got %d", len(cache.invalidateCalls))
	}
}

func TestHandleStatusUpdateUseCase_Execute_Failed(t *testing.T) {
	repo := newMockRepo()
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewHandleStatusUpdateUseCase(repo, cache, logger)
	err := uc.Execute(context.Background(), entity.VideoStatusMessage{
		JobID:        v.ID,
		UserID:       "user-1",
		Status:       "FAILED",
		ErrorMessage: "ffmpeg crashed",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	updated := repo.videos[v.ID]
	if updated.Status != entity.VideoStatusFailed {
		t.Errorf("expected FAILED, got %s", updated.Status)
	}
	if updated.ErrorMessage != "ffmpeg crashed" {
		t.Errorf("expected ErrorMessage=ffmpeg crashed, got %s", updated.ErrorMessage)
	}
}

func TestHandleStatusUpdateUseCase_Execute_UnknownStatusIgnored(t *testing.T) {
	repo := newMockRepo()
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewHandleStatusUpdateUseCase(repo, cache, logger)
	err := uc.Execute(context.Background(), entity.VideoStatusMessage{
		JobID:  v.ID,
		UserID: "user-1",
		Status: "UNKNOWN_STATUS",
	})

	if err != nil {
		t.Fatalf("expected success for unknown status (ignored), got: %v", err)
	}
	// Status should remain unchanged
	if repo.videos[v.ID].Status != entity.VideoStatusPending {
		t.Errorf("expected status to remain PENDING, got %s", repo.videos[v.ID].Status)
	}
}

func TestHandleStatusUpdateUseCase_Execute_VideoNotFound(t *testing.T) {
	repo := newMockRepo()
	repo.findErr = errors.New("not found")
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewHandleStatusUpdateUseCase(repo, cache, logger)
	err := uc.Execute(context.Background(), entity.VideoStatusMessage{
		JobID:  [16]byte{1},
		Status: "COMPLETED",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleStatusUpdateUseCase_Execute_UpdateError(t *testing.T) {
	repo := newMockRepo()
	repo.updateErr = errors.New("db error")
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewHandleStatusUpdateUseCase(repo, cache, logger)
	err := uc.Execute(context.Background(), entity.VideoStatusMessage{
		JobID:  v.ID,
		Status: "COMPLETED",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleStatusUpdateUseCase_Execute_CacheErrorIgnored(t *testing.T) {
	repo := newMockRepo()
	cache := newMockCache()
	cache.invalidateErr = errors.New("redis down")
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewHandleStatusUpdateUseCase(repo, cache, logger)
	err := uc.Execute(context.Background(), entity.VideoStatusMessage{
		JobID:  v.ID,
		Status: "COMPLETED",
	})

	if err != nil {
		t.Fatalf("expected success despite cache error, got: %v", err)
	}
}

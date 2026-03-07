package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

func TestListVideosUseCase_Execute_FromCache(t *testing.T) {
	repo := newMockRepo()
	cache := newMockCache()
	logger := zap.NewNop()

	cached := []*entity.Video{
		{UserID: "user-1", OriginalName: "cached.mp4"},
	}
	cache.store["user-1"] = cached

	uc := usecase.NewListVideosUseCase(repo, cache, logger)
	videos, err := uc.Execute(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(videos))
	}
}

func TestListVideosUseCase_Execute_FromRepo(t *testing.T) {
	repo := newMockRepo()
	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	cache := newMockCache()
	cache.getErr = errors.New("cache miss")
	logger := zap.NewNop()

	uc := usecase.NewListVideosUseCase(repo, cache, logger)
	videos, err := uc.Execute(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(videos))
	}
	// Should have cached the result
	if _, ok := cache.store["user-1"]; !ok {
		t.Error("expected result to be cached")
	}
}

func TestListVideosUseCase_Execute_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.findErr = errors.New("db error")

	cache := newMockCache()
	cache.getErr = errors.New("cache miss")
	logger := zap.NewNop()

	uc := usecase.NewListVideosUseCase(repo, cache, logger)
	_, err := uc.Execute(context.Background(), "user-1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListVideosUseCase_Execute_CacheSetErrorIgnored(t *testing.T) {
	repo := newMockRepo()
	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	cache := newMockCache()
	cache.getErr = errors.New("cache miss")
	cache.setErr = errors.New("redis down")
	logger := zap.NewNop()

	uc := usecase.NewListVideosUseCase(repo, cache, logger)
	videos, err := uc.Execute(context.Background(), "user-1")

	// Should succeed despite cache set error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(videos))
	}
}

func TestListVideosUseCase_Execute_EmptyList(t *testing.T) {
	repo := newMockRepo()
	cache := newMockCache()
	cache.getErr = errors.New("cache miss")
	logger := zap.NewNop()

	uc := usecase.NewListVideosUseCase(repo, cache, logger)
	videos, err := uc.Execute(context.Background(), "user-nobody")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if videos == nil {
		// nil is acceptable — handler converts to empty slice
		return
	}
	if len(videos) != 0 {
		t.Errorf("expected 0 videos, got %d", len(videos))
	}
}

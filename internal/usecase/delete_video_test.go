package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

func TestDeleteVideoUseCase_Execute_Success(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewDeleteVideoUseCase(repo, storage, cache, logger)
	err := uc.Execute(context.Background(), v.ID, "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.videos[v.ID]; ok {
		t.Error("expected video to be deleted from repo")
	}
	if len(cache.invalidateCalls) != 1 {
		t.Errorf("expected 1 cache invalidation, got %d", len(cache.invalidateCalls))
	}
}

func TestDeleteVideoUseCase_Execute_NotFound(t *testing.T) {
	repo := newMockRepo()
	repo.findErr = errors.New("not found")
	storage := &mockStorage{}
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewDeleteVideoUseCase(repo, storage, cache, logger)
	err := uc.Execute(context.Background(), [16]byte{1}, "user-1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestDeleteVideoUseCase_Execute_WrongUser(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewDeleteVideoUseCase(repo, storage, cache, logger)
	err := uc.Execute(context.Background(), v.ID, "user-attacker")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestDeleteVideoUseCase_Execute_StorageErrorIgnored(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{deleteErr: errors.New("storage error")}
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewDeleteVideoUseCase(repo, storage, cache, logger)
	// Storage delete error should be logged but not returned
	err := uc.Execute(context.Background(), v.ID, "user-1")

	if err != nil {
		t.Fatalf("expected success despite storage error, got: %v", err)
	}
}

func TestDeleteVideoUseCase_Execute_RepoDeleteError(t *testing.T) {
	repo := newMockRepo()
	repo.deleteErr = errors.New("db error")
	storage := &mockStorage{}
	cache := newMockCache()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewDeleteVideoUseCase(repo, storage, cache, logger)
	err := uc.Execute(context.Background(), v.ID, "user-1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteVideoUseCase_Execute_CacheErrorIgnored(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	cache := newMockCache()
	cache.invalidateErr = errors.New("redis down")
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewDeleteVideoUseCase(repo, storage, cache, logger)
	err := uc.Execute(context.Background(), v.ID, "user-1")

	if err != nil {
		t.Fatalf("expected success despite cache error, got: %v", err)
	}
}

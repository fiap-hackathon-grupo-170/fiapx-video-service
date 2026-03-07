package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

func TestUploadVideoUseCase_Execute_Success(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	publisher := &mockPublisher{}
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)

	reader := strings.NewReader("fake video content")
	video, err := uc.Execute(context.Background(), reader, "test.mp4", 18, "user-1", "user@example.com")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if video == nil {
		t.Fatal("expected video, got nil")
	}
	if video.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", video.UserID)
	}
	if video.OriginalName != "test.mp4" {
		t.Errorf("expected OriginalName=test.mp4, got %s", video.OriginalName)
	}
	if len(publisher.publishedMsgs) != 1 {
		t.Errorf("expected 1 published message, got %d", len(publisher.publishedMsgs))
	}
	if len(cache.invalidateCalls) != 1 {
		t.Errorf("expected 1 cache invalidation, got %d", len(cache.invalidateCalls))
	}
}

func TestUploadVideoUseCase_Execute_StorageError(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{uploadErr: errors.New("minio unavailable")}
	publisher := &mockPublisher{}
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)

	_, err := uc.Execute(context.Background(), strings.NewReader("data"), "test.mp4", 4, "user-1", "user@example.com")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadVideoUseCase_Execute_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = errors.New("db error")
	storage := &mockStorage{}
	publisher := &mockPublisher{}
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)

	_, err := uc.Execute(context.Background(), strings.NewReader("data"), "test.mp4", 4, "user-1", "user@example.com")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadVideoUseCase_Execute_PublisherError(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	publisher := &mockPublisher{publishErr: errors.New("rabbitmq down")}
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)

	_, err := uc.Execute(context.Background(), strings.NewReader("data"), "test.mp4", 4, "user-1", "user@example.com")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUploadVideoUseCase_Execute_CacheErrorIgnored(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	publisher := &mockPublisher{}
	cache := newMockCache()
	cache.invalidateErr = errors.New("redis down")
	logger := zap.NewNop()

	uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)

	// Cache error should be ignored; upload still succeeds
	video, err := uc.Execute(context.Background(), strings.NewReader("data"), "test.mp4", 4, "user-1", "user@example.com")

	if err != nil {
		t.Fatalf("expected success despite cache error, got: %v", err)
	}
	if video == nil {
		t.Fatal("expected video result")
	}
}

func TestUploadVideoUseCase_Execute_VideoKeyContainsUserID(t *testing.T) {
	repo := newMockRepo()
	storage := &mockStorage{}
	publisher := &mockPublisher{}
	cache := newMockCache()
	logger := zap.NewNop()

	uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)

	video, err := uc.Execute(context.Background(), strings.NewReader("data"), "myvideo.mp4", 4, "user-42", "u@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(video.VideoKey, "user-42/") {
		t.Errorf("expected VideoKey to start with user-42/, got %s", video.VideoKey)
	}
}

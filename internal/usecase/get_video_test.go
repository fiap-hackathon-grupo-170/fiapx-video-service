package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

func TestGetVideoUseCase_Execute_Success(t *testing.T) {
	repo := newMockRepo()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewGetVideoUseCase(repo, logger)
	result, err := uc.Execute(context.Background(), v.ID, "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != v.ID {
		t.Errorf("expected video ID %s, got %s", v.ID, result.ID)
	}
}

func TestGetVideoUseCase_Execute_NotFound(t *testing.T) {
	repo := newMockRepo()
	repo.findErr = errors.New("not found")
	logger := zap.NewNop()

	uc := usecase.NewGetVideoUseCase(repo, logger)

	_, err := uc.Execute(context.Background(), [16]byte{1}, "user-1")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, usecase.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

func TestGetVideoUseCase_Execute_WrongUser(t *testing.T) {
	repo := newMockRepo()
	logger := zap.NewNop()

	v := entity.NewVideo("user-1", "u@e.com", "key.mp4", "video.mp4", 100)
	repo.videos[v.ID] = v

	uc := usecase.NewGetVideoUseCase(repo, logger)
	_, err := uc.Execute(context.Background(), v.ID, "user-attacker")

	if err == nil {
		t.Fatal("expected error for wrong user, got nil")
	}
	if !errors.Is(err, usecase.ErrVideoNotFound) {
		t.Errorf("expected ErrVideoNotFound, got %v", err)
	}
}

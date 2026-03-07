package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"go.uber.org/zap"
)

// TestUploadVideoUseCase_ContentTypes exercises the videoContentType helper
// by uploading files with different extensions and verifying success.
func TestUploadVideoUseCase_ContentTypes(t *testing.T) {
	extensions := []string{
		"video.mp4",
		"video.webm",
		"video.avi",
		"video.mov",
		"video.mkv",
		"video.wmv",
		"video.flv",
	}

	for _, name := range extensions {
		t.Run(name, func(t *testing.T) {
			repo := newMockRepo()
			storage := &mockStorage{}
			publisher := &mockPublisher{}
			cache := newMockCache()
			logger := zap.NewNop()

			uc := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, logger)
			_, err := uc.Execute(context.Background(), strings.NewReader("data"), name, 4, "user-1", "u@e.com")
			if err != nil {
				t.Errorf("unexpected error for %s: %v", name, err)
			}
		})
	}
}

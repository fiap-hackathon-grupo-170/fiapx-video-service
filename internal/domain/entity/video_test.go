package entity_test

import (
	"testing"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
)

func TestNewVideo(t *testing.T) {
	before := time.Now().UTC()
	v := entity.NewVideo("user-1", "user@example.com", "user-1/abc.mp4", "video.mp4", 1024)
	after := time.Now().UTC()

	if v.UserID != "user-1" {
		t.Errorf("expected UserID=user-1, got %s", v.UserID)
	}
	if v.UserEmail != "user@example.com" {
		t.Errorf("expected UserEmail=user@example.com, got %s", v.UserEmail)
	}
	if v.VideoKey != "user-1/abc.mp4" {
		t.Errorf("expected VideoKey=user-1/abc.mp4, got %s", v.VideoKey)
	}
	if v.OriginalName != "video.mp4" {
		t.Errorf("expected OriginalName=video.mp4, got %s", v.OriginalName)
	}
	if v.FileSize != 1024 {
		t.Errorf("expected FileSize=1024, got %d", v.FileSize)
	}
	if v.Status != entity.VideoStatusPending {
		t.Errorf("expected Status=PENDING, got %s", v.Status)
	}
	if v.ID == [16]byte{} {
		t.Error("expected non-zero UUID")
	}
	if v.CreatedAt.Before(before) || v.CreatedAt.After(after) {
		t.Error("CreatedAt not in expected range")
	}
	if v.CompletedAt != nil {
		t.Error("expected CompletedAt to be nil")
	}
}

func TestVideo_MarkProcessing(t *testing.T) {
	v := entity.NewVideo("user-1", "user@example.com", "key", "video.mp4", 1024)
	before := time.Now().UTC()
	v.MarkProcessing()

	if v.Status != entity.VideoStatusProcessing {
		t.Errorf("expected Status=PROCESSING, got %s", v.Status)
	}
	if v.UpdatedAt.Before(before) {
		t.Error("UpdatedAt not updated")
	}
}

func TestVideo_MarkCompleted(t *testing.T) {
	v := entity.NewVideo("user-1", "user@example.com", "key", "video.mp4", 1024)
	before := time.Now().UTC()
	v.MarkCompleted("zip/frames.zip", 42, 3.5)

	if v.Status != entity.VideoStatusCompleted {
		t.Errorf("expected Status=COMPLETED, got %s", v.Status)
	}
	if v.ZipKey != "zip/frames.zip" {
		t.Errorf("expected ZipKey=zip/frames.zip, got %s", v.ZipKey)
	}
	if v.FrameCount != 42 {
		t.Errorf("expected FrameCount=42, got %d", v.FrameCount)
	}
	if v.VideoDuration != 3.5 {
		t.Errorf("expected VideoDuration=3.5, got %f", v.VideoDuration)
	}
	if v.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
	if v.CompletedAt.Before(before) {
		t.Error("CompletedAt not in expected range")
	}
}

func TestVideo_MarkFailed(t *testing.T) {
	v := entity.NewVideo("user-1", "user@example.com", "key", "video.mp4", 1024)
	before := time.Now().UTC()
	v.MarkFailed("processing error occurred")

	if v.Status != entity.VideoStatusFailed {
		t.Errorf("expected Status=FAILED, got %s", v.Status)
	}
	if v.ErrorMessage != "processing error occurred" {
		t.Errorf("expected ErrorMessage=processing error occurred, got %s", v.ErrorMessage)
	}
	if v.UpdatedAt.Before(before) {
		t.Error("UpdatedAt not updated")
	}
	if v.CompletedAt != nil {
		t.Error("expected CompletedAt to remain nil")
	}
}

func TestVideoStatusConstants(t *testing.T) {
	if entity.VideoStatusPending != "PENDING" {
		t.Error("VideoStatusPending should be PENDING")
	}
	if entity.VideoStatusProcessing != "PROCESSING" {
		t.Error("VideoStatusProcessing should be PROCESSING")
	}
	if entity.VideoStatusCompleted != "COMPLETED" {
		t.Error("VideoStatusCompleted should be COMPLETED")
	}
	if entity.VideoStatusFailed != "FAILED" {
		t.Error("VideoStatusFailed should be FAILED")
	}
}

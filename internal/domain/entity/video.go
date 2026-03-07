package entity

import (
	"time"

	"github.com/google/uuid"
)

type VideoStatus string

const (
	VideoStatusPending    VideoStatus = "PENDING"
	VideoStatusProcessing VideoStatus = "PROCESSING"
	VideoStatusCompleted  VideoStatus = "COMPLETED"
	VideoStatusFailed     VideoStatus = "FAILED"
)

type Video struct {
	ID            uuid.UUID   `json:"id"`
	UserID        string      `json:"user_id"`
	UserEmail     string      `json:"user_email"`
	VideoKey      string      `json:"video_key"`
	ZipKey        string      `json:"zip_key,omitempty"`
	OriginalName  string      `json:"original_name"`
	FileSize      int64       `json:"file_size"`
	Status        VideoStatus `json:"status"`
	FrameCount    int         `json:"frame_count"`
	VideoDuration float64     `json:"video_duration"`
	ErrorMessage  string      `json:"error_message,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
}

func NewVideo(userID, userEmail, videoKey, originalName string, fileSize int64) *Video {
	now := time.Now().UTC()
	return &Video{
		ID:           uuid.New(),
		UserID:       userID,
		UserEmail:    userEmail,
		VideoKey:     videoKey,
		OriginalName: originalName,
		FileSize:     fileSize,
		Status:       VideoStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (v *Video) MarkProcessing() {
	v.Status = VideoStatusProcessing
	v.UpdatedAt = time.Now().UTC()
}

func (v *Video) MarkCompleted(zipKey string, frameCount int, duration float64) {
	now := time.Now().UTC()
	v.Status = VideoStatusCompleted
	v.ZipKey = zipKey
	v.FrameCount = frameCount
	v.VideoDuration = duration
	v.UpdatedAt = now
	v.CompletedAt = &now
}

func (v *Video) MarkFailed(errMsg string) {
	v.Status = VideoStatusFailed
	v.ErrorMessage = errMsg
	v.UpdatedAt = time.Now().UTC()
}

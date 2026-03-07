package entity

import "github.com/google/uuid"

// VideoUploadedMessage is published to the video.processing queue.
type VideoUploadedMessage struct {
	JobID     uuid.UUID `json:"job_id"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	VideoKey  string    `json:"video_key"`
	FileSize  int64     `json:"file_size"`
}

// VideoStatusMessage is consumed from the video.status queue.
type VideoStatusMessage struct {
	JobID        uuid.UUID `json:"job_id"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	VideoKey     string    `json:"video_key"`
	ZipKey       string    `json:"zip_key,omitempty"`
	FrameCount   int       `json:"frame_count,omitempty"`
	Duration     float64   `json:"duration_seconds,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Attempt      int       `json:"attempt"`
	MaxAttempts  int       `json:"max_attempts"`
}

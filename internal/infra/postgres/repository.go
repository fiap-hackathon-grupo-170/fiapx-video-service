package postgres

import (
	"context"
	"fmt"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dbPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type VideoRepository struct {
	pool dbPool
}

func NewVideoRepository(pool *pgxpool.Pool) *VideoRepository {
	return &VideoRepository{pool: pool}
}

func (r *VideoRepository) Create(ctx context.Context, video *entity.Video) error {
	query := `
		INSERT INTO videos (
			id, user_id, user_email, video_key, zip_key, original_name, status,
			frame_count, file_size, video_duration, error_message,
			created_at, updated_at, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`

	_, err := r.pool.Exec(ctx, query,
		video.ID, video.UserID, video.UserEmail, video.VideoKey, video.ZipKey,
		video.OriginalName, string(video.Status),
		video.FrameCount, video.FileSize, video.VideoDuration, video.ErrorMessage,
		video.CreatedAt, video.UpdatedAt, video.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert video: %w", err)
	}
	return nil
}

func (r *VideoRepository) Update(ctx context.Context, video *entity.Video) error {
	query := `
		UPDATE videos SET
			status=$2, zip_key=$3, frame_count=$4, video_duration=$5,
			error_message=$6, updated_at=$7, completed_at=$8
		WHERE id=$1`

	_, err := r.pool.Exec(ctx, query,
		video.ID, string(video.Status), video.ZipKey, video.FrameCount,
		video.VideoDuration, video.ErrorMessage,
		video.UpdatedAt, video.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update video: %w", err)
	}
	return nil
}

func (r *VideoRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Video, error) {
	query := `
		SELECT id, user_id, user_email, video_key, zip_key, original_name, status,
			frame_count, file_size, video_duration, error_message,
			created_at, updated_at, completed_at
		FROM videos WHERE id=$1`

	video := &entity.Video{}
	var status string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&video.ID, &video.UserID, &video.UserEmail, &video.VideoKey, &video.ZipKey,
		&video.OriginalName, &status,
		&video.FrameCount, &video.FileSize, &video.VideoDuration, &video.ErrorMessage,
		&video.CreatedAt, &video.UpdatedAt, &video.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("find video by id: %w", err)
	}
	video.Status = entity.VideoStatus(status)
	return video, nil
}

func (r *VideoRepository) FindByUserID(ctx context.Context, userID string) ([]*entity.Video, error) {
	query := `
		SELECT id, user_id, user_email, video_key, zip_key, original_name, status,
			frame_count, file_size, video_duration, error_message,
			created_at, updated_at, completed_at
		FROM videos WHERE user_id=$1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query videos by user: %w", err)
	}
	defer rows.Close()

	var videos []*entity.Video
	for rows.Next() {
		video := &entity.Video{}
		var status string
		if err := rows.Scan(
			&video.ID, &video.UserID, &video.UserEmail, &video.VideoKey, &video.ZipKey,
			&video.OriginalName, &status,
			&video.FrameCount, &video.FileSize, &video.VideoDuration, &video.ErrorMessage,
			&video.CreatedAt, &video.UpdatedAt, &video.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan video row: %w", err)
		}
		video.Status = entity.VideoStatus(status)
		videos = append(videos, video)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate video rows: %w", err)
	}

	return videos, nil
}

func (r *VideoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM videos WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete video: %w", err)
	}
	return nil
}

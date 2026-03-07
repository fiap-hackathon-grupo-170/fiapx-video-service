package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	httpinfra "github.com/fiapx/fiapx-video-service/internal/infra/http"
	minioinfra "github.com/fiapx/fiapx-video-service/internal/infra/minio"
	"github.com/fiapx/fiapx-video-service/internal/infra/postgres"
	"github.com/fiapx/fiapx-video-service/internal/infra/rabbitmq"
	redisinfra "github.com/fiapx/fiapx-video-service/internal/infra/redis"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcminio "github.com/testcontainers/testcontainers-go/modules/minio"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"go.uber.org/zap"
)

// staticTokenValidator always returns a fixed user for tests.
type staticTokenValidator struct{}

func (v *staticTokenValidator) Validate(tokenString string) (*port.Claims, error) {
	return &port.Claims{
		UserID:    "test-user-id",
		UserEmail: "test@fiapx.local",
	}, nil
}

func TestUploadAndListFlow(t *testing.T) {
	ctx := context.Background()
	log := zap.NewNop()

	// --- Start containers ---
	pgContainer, err := tcpostgres.Run(ctx, "postgres:15-alpine",
		tcpostgres.WithDatabase("videos"),
		tcpostgres.WithUsername("video_user"),
		tcpostgres.WithPassword("video_pass"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { pgContainer.Terminate(ctx) })

	minioContainer, err := tcminio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	require.NoError(t, err)
	t.Cleanup(func() { minioContainer.Terminate(ctx) })

	rmqContainer, err := tcrabbitmq.Run(ctx, "rabbitmq:3.12-management-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { rmqContainer.Terminate(ctx) })

	// --- PostgreSQL ---
	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, pgDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, postgres.RunMigrations(pgDSN, "../../migrations"))

	// --- MinIO ---
	minioEndpoint, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err)

	storage, err := minioinfra.NewStorage(minioinfra.StorageConfig{
		Endpoint:     minioEndpoint,
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		UploadBucket: "uploads",
		ZipBucket:    "zips",
	})
	require.NoError(t, err)
	require.NoError(t, storage.EnsureBuckets(ctx))

	// --- RabbitMQ ---
	rmqURL, err := rmqContainer.AmqpURL(ctx)
	require.NoError(t, err)

	rmqConn, err := amqp.Dial(rmqURL)
	require.NoError(t, err)
	t.Cleanup(func() { rmqConn.Close() })

	publisher, err := rabbitmq.NewPublisher(rmqConn, "fiapx.video")
	require.NoError(t, err)

	// Declare the processing queue so messages can be inspected in assertions
	ch, err := rmqConn.Channel()
	require.NoError(t, err)
	t.Cleanup(func() { ch.Close() })
	_, err = ch.QueueDeclare("video.processing", true, false, false, false, nil)
	require.NoError(t, err)
	err = ch.QueueBind("video.processing", "video.processing", "fiapx.video", false, nil)
	require.NoError(t, err)

	// --- Redis (optional; non-existent port causes cache miss gracefully) ---
	cache, err := redisinfra.NewVideoCache("redis://localhost:16399/0")
	require.NoError(t, err)

	// --- Use Cases ---
	repo := postgres.NewVideoRepository(pool)
	uploadUC := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, log)
	listUC := usecase.NewListVideosUseCase(repo, cache, log)
	getUC := usecase.NewGetVideoUseCase(repo, log)
	downloadUC := usecase.NewDownloadVideoUseCase(repo, storage, log)
	deleteUC := usecase.NewDeleteVideoUseCase(repo, storage, cache, log)

	// --- HTTP Server ---
	handler := httpinfra.NewHandler(uploadUC, listUC, getUC, downloadUC, deleteUC, storage, log, 500)
	server := httpinfra.NewServer(handler, &staticTokenValidator{}, 0, log)
	ts := httptest.NewServer(server.Engine())
	t.Cleanup(ts.Close)

	var videoID string

	// --- Test: Upload ---
	t.Run("upload video", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("video", "test.mp4")
		require.NoError(t, err)
		_, err = io.WriteString(part, "fake video content for test")
		require.NoError(t, err)
		writer.Close()

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/videos/upload", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, "PENDING", result["status"])
		assert.NotEmpty(t, result["id"])

		videoID = result["id"].(string)

		// Verify record in DB
		time.Sleep(100 * time.Millisecond)
		videos, err := repo.FindByUserID(ctx, "test-user-id")
		require.NoError(t, err)
		assert.Len(t, videos, 1)
		assert.Equal(t, entity.VideoStatusPending, videos[0].Status)

		// Verify message in RabbitMQ
		msgs, err := ch.Consume("video.processing", "", true, false, false, false, nil)
		require.NoError(t, err)

		select {
		case msg := <-msgs:
			var uploaded entity.VideoUploadedMessage
			require.NoError(t, json.Unmarshal(msg.Body, &uploaded))
			assert.Equal(t, videoID, uploaded.JobID.String())
			assert.Equal(t, "test-user-id", uploaded.UserID)
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for RabbitMQ message")
		}
	})

	// --- Test: List ---
	t.Run("list videos", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, float64(1), result["total"])
	})

	// --- Test: Get video ---
	t.Run("get video", func(t *testing.T) {
		require.NotEmpty(t, videoID)

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos/"+videoID, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// --- Test: Unauthorized request ---
	t.Run("unauthorized without token", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/videos")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestHealthEndpoint(t *testing.T) {
	log := zap.NewNop()

	handler := httpinfra.NewHandler(
		usecase.NewUploadVideoUseCase(&noopRepo{}, &noopStorage{}, &noopPublisher{}, &noopCache{}, log),
		usecase.NewListVideosUseCase(&noopRepo{}, &noopCache{}, log),
		usecase.NewGetVideoUseCase(&noopRepo{}, log),
		usecase.NewDownloadVideoUseCase(&noopRepo{}, &noopStorage{}, log),
		usecase.NewDeleteVideoUseCase(&noopRepo{}, &noopStorage{}, &noopCache{}, log),
		&noopStorage{},
		log,
		500,
	)
	server := httpinfra.NewServer(handler, &staticTokenValidator{}, 0, log)
	ts := httptest.NewServer(server.Engine())
	defer ts.Close()

	resp, err := http.Get(fmt.Sprintf("%s/health", ts.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// --- no-op implementations for unit-level tests ---

type noopRepo struct{}

func (r *noopRepo) Create(ctx context.Context, v *entity.Video) error { return nil }
func (r *noopRepo) Update(ctx context.Context, v *entity.Video) error { return nil }
func (r *noopRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Video, error) {
	return nil, fmt.Errorf("not found")
}
func (r *noopRepo) FindByUserID(ctx context.Context, userID string) ([]*entity.Video, error) {
	return nil, nil
}
func (r *noopRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type noopStorage struct{}

func (s *noopStorage) UploadVideo(ctx context.Context, key string, reader io.Reader, size int64, ct string) error {
	return nil
}
func (s *noopStorage) DeleteVideo(ctx context.Context, key string) error { return nil }
func (s *noopStorage) PresignedDownloadURL(ctx context.Context, zipKey string, ttl time.Duration) (string, error) {
	return "https://example.com/presigned", nil
}
func (s *noopStorage) StreamZip(ctx context.Context, zipKey string) (io.ReadCloser, int64, error) {
	return io.NopCloser(bytes.NewReader(nil)), 0, nil
}

type noopPublisher struct{}

func (p *noopPublisher) PublishVideoUploaded(ctx context.Context, msg entity.VideoUploadedMessage) error {
	return nil
}

type noopCache struct{}

func (c *noopCache) GetUserVideos(ctx context.Context, userID string) ([]*entity.Video, error) {
	return nil, fmt.Errorf("cache miss")
}
func (c *noopCache) SetUserVideos(ctx context.Context, userID string, videos []*entity.Video, ttl time.Duration) error {
	return nil
}
func (c *noopCache) InvalidateUserVideos(ctx context.Context, userID string) error { return nil }

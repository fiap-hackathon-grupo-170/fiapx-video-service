package http_test

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
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type testRepo struct {
	videos    map[uuid.UUID]*entity.Video
	createErr error
	updateErr error
	findErr   error
	deleteErr error
}

func newTestRepo() *testRepo {
	return &testRepo{videos: make(map[uuid.UUID]*entity.Video)}
}

func (r *testRepo) Create(_ context.Context, v *entity.Video) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.videos[v.ID] = v
	return nil
}

func (r *testRepo) Update(_ context.Context, v *entity.Video) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.videos[v.ID] = v
	return nil
}

func (r *testRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.Video, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	v, ok := r.videos[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return v, nil
}

func (r *testRepo) FindByUserID(_ context.Context, userID string) ([]*entity.Video, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	var result []*entity.Video
	for _, v := range r.videos {
		if v.UserID == userID {
			result = append(result, v)
		}
	}
	return result, nil
}

func (r *testRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.videos, id)
	return nil
}

type testStorage struct {
	uploadErr    error
	deleteErr    error
	presignURL   string
	presignErr   error
	streamReader io.ReadCloser
	streamSize   int64
	streamErr    error
}

func (s *testStorage) UploadVideo(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return s.uploadErr
}

func (s *testStorage) DeleteVideo(_ context.Context, _ string) error {
	return s.deleteErr
}

func (s *testStorage) PresignedDownloadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	if s.presignErr != nil {
		return "", s.presignErr
	}
	return s.presignURL, nil
}

func (s *testStorage) StreamZip(_ context.Context, _ string) (io.ReadCloser, int64, error) {
	if s.streamErr != nil {
		return nil, 0, s.streamErr
	}
	if s.streamReader != nil {
		return s.streamReader, s.streamSize, nil
	}
	return io.NopCloser(bytes.NewReader([]byte("zip content"))), 11, nil
}

type testPublisher struct {
	publishErr error
}

func (p *testPublisher) PublishVideoUploaded(_ context.Context, _ entity.VideoUploadedMessage) error {
	return p.publishErr
}

type testCache struct {
	getErr        error
	invalidateErr error
}

func (c *testCache) GetUserVideos(_ context.Context, _ string) ([]*entity.Video, error) {
	return nil, c.getErr
}

func (c *testCache) SetUserVideos(_ context.Context, _ string, _ []*entity.Video, _ time.Duration) error {
	return nil
}

func (c *testCache) InvalidateUserVideos(_ context.Context, _ string) error {
	return c.invalidateErr
}

type testValidator struct {
	claims *port.Claims
	err    error
}

func (v *testValidator) Validate(_ string) (*port.Claims, error) {
	return v.claims, v.err
}

// --- Test helpers ---

func buildServer(repo *testRepo, storage *testStorage, publisher *testPublisher, cache *testCache, validator port.TokenValidator) *httptest.Server {
	log := zap.NewNop()
	uploadUC := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, log)
	listUC := usecase.NewListVideosUseCase(repo, cache, log)
	getUC := usecase.NewGetVideoUseCase(repo, log)
	downloadUC := usecase.NewDownloadVideoUseCase(repo, storage, log)
	deleteUC := usecase.NewDeleteVideoUseCase(repo, storage, cache, log)

	if validator == nil {
		validator = &testValidator{
			claims: &port.Claims{UserID: "user-1", UserEmail: "user@test.com"},
		}
	}

	handler := httpinfra.NewHandler(uploadUC, listUC, getUC, downloadUC, deleteUC, storage, log, 100)
	srv := httpinfra.NewServer(handler, validator, 0, log)
	return httptest.NewServer(srv.Engine())
}

func authedRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	return req
}

// --- Tests ---

func TestListHandler_EmptyList(t *testing.T) {
	repo := newTestRepo()
	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(0), result["total"])
}

func TestListHandler_RepoError(t *testing.T) {
	repo := newTestRepo()
	repo.findErr = fmt.Errorf("db error")
	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestGetHandler_NotFound(t *testing.T) {
	repo := newTestRepo()
	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	id := uuid.New()
	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+id.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestGetHandler_InvalidID(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/invalid-uuid", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetHandler_Found(t *testing.T) {
	repo := newTestRepo()
	video := entity.NewVideo("user-1", "user@test.com", "videos/test.mp4", "test.mp4", 1024)
	repo.videos[video.ID] = video

	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+video.ID.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetHandler_WrongUser(t *testing.T) {
	repo := newTestRepo()
	video := entity.NewVideo("other-user", "other@test.com", "videos/test.mp4", "test.mp4", 1024)
	repo.videos[video.ID] = video

	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+video.ID.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUploadHandler_InvalidExtension(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("video", "test.txt")
	require.NoError(t, err)
	_, _ = io.WriteString(part, "fake content")
	writer.Close()

	req := authedRequest(t, http.MethodPost, ts.URL+"/api/videos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUploadHandler_ValidMP4(t *testing.T) {
	repo := newTestRepo()
	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, nil)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("video", "video.mp4")
	require.NoError(t, err)
	_, _ = io.WriteString(part, "fake mp4 content")
	writer.Close()

	req := authedRequest(t, http.MethodPost, ts.URL+"/api/videos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestUploadHandler_StorageError(t *testing.T) {
	storage := &testStorage{uploadErr: fmt.Errorf("storage failure")}
	ts := buildServer(newTestRepo(), storage, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, nil)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("video", "video.mp4")
	require.NoError(t, err)
	_, _ = io.WriteString(part, "fake mp4 content")
	writer.Close()

	req := authedRequest(t, http.MethodPost, ts.URL+"/api/videos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestUploadHandler_NoFile(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req := authedRequest(t, http.MethodPost, ts.URL+"/api/videos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteHandler_InvalidID(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodDelete, ts.URL+"/api/videos/invalid", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteHandler_NotFound(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	id := uuid.New()
	req := authedRequest(t, http.MethodDelete, ts.URL+"/api/videos/"+id.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteHandler_Success(t *testing.T) {
	repo := newTestRepo()
	video := entity.NewVideo("user-1", "user@test.com", "videos/test.mp4", "test.mp4", 1024)
	repo.videos[video.ID] = video

	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodDelete, ts.URL+"/api/videos/"+video.ID.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestDownloadHandler_InvalidID(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/bad-id/download", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDownloadHandler_NotFound(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	id := uuid.New()
	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+id.String()+"/download", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDownloadHandler_NotReady(t *testing.T) {
	repo := newTestRepo()
	video := entity.NewVideo("user-1", "user@test.com", "videos/test.mp4", "test.mp4", 1024)
	// Status is PENDING (not completed), so download should fail with not ready
	repo.videos[video.ID] = video

	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+video.ID.String()+"/download", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestDownloadHandler_Success(t *testing.T) {
	repo := newTestRepo()
	video := entity.NewVideo("user-1", "user@test.com", "videos/test.mp4", "test.mp4", 1024)
	video.MarkCompleted("zips/test.zip", 10, 5.0)
	repo.videos[video.ID] = video

	ts := buildServer(repo, &testStorage{presignURL: "http://minio/zip"}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+video.ID.String()+"/download", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/zip", resp.Header.Get("Content-Type"))
}

func TestDownloadHandler_FailedVideo(t *testing.T) {
	repo := newTestRepo()
	video := entity.NewVideo("user-1", "user@test.com", "videos/test.mp4", "test.mp4", 1024)
	video.MarkFailed("processing error")
	repo.videos[video.ID] = video

	ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req := authedRequest(t, http.MethodGet, ts.URL+"/api/videos/"+video.ID.String()+"/download", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestHealthEndpoint(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCORSPreflight(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodOptions, ts.URL+"/api/videos", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

func TestUploadHandler_AllValidExtensions(t *testing.T) {
	extensions := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm"}
	for _, ext := range extensions {
		t.Run(ext, func(t *testing.T) {
			repo := newTestRepo()
			ts := buildServer(repo, &testStorage{}, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, nil)
			defer ts.Close()

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("video", "video"+ext)
			require.NoError(t, err)
			_, _ = io.WriteString(part, "fake content")
			writer.Close()

			req := authedRequest(t, http.MethodPost, ts.URL+"/api/videos/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		})
	}
}

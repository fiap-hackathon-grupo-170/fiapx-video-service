package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	httpinfra "github.com/fiapx/fiapx-video-service/internal/infra/http"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewServer_HealthRoute(t *testing.T) {
	log := zap.NewNop()
	repo := newTestRepo()
	storage := &testStorage{}
	publisher := &testPublisher{}
	cache := &testCache{}

	uploadUC := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, log)
	listUC := usecase.NewListVideosUseCase(repo, cache, log)
	getUC := usecase.NewGetVideoUseCase(repo, log)
	downloadUC := usecase.NewDownloadVideoUseCase(repo, storage, log)
	deleteUC := usecase.NewDeleteVideoUseCase(repo, storage, cache, log)

	validator := &testValidator{claims: &port.Claims{UserID: "u1", UserEmail: "u@test.com"}}
	handler := httpinfra.NewHandler(uploadUC, listUC, getUC, downloadUC, deleteUC, storage, log, 500)
	srv := httpinfra.NewServer(handler, validator, 0, log)

	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewServer_Engine_NotNil(t *testing.T) {
	log := zap.NewNop()
	repo := newTestRepo()
	storage := &testStorage{}
	publisher := &testPublisher{}
	cache := &testCache{}

	uploadUC := usecase.NewUploadVideoUseCase(repo, storage, publisher, cache, log)
	listUC := usecase.NewListVideosUseCase(repo, cache, log)
	getUC := usecase.NewGetVideoUseCase(repo, log)
	downloadUC := usecase.NewDownloadVideoUseCase(repo, storage, log)
	deleteUC := usecase.NewDeleteVideoUseCase(repo, storage, cache, log)

	validator := &testValidator{claims: &port.Claims{UserID: "u1", UserEmail: "u@test.com"}}
	handler := httpinfra.NewHandler(uploadUC, listUC, getUC, downloadUC, deleteUC, storage, log, 500)
	srv := httpinfra.NewServer(handler, validator, 8099, log)

	assert.NotNil(t, srv.Engine())
}

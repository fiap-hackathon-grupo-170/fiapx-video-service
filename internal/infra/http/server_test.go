package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainport "github.com/fiapx/fiapx-video-service/internal/domain/port"
	httpinfra "github.com/fiapx/fiapx-video-service/internal/infra/http"
	"github.com/fiapx/fiapx-video-service/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func makeTestServer(port int) *httpinfra.Server {
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

	validator := &testValidator{claims: &domainport.Claims{UserID: "u1", UserEmail: "u@test.com"}}
	handler := httpinfra.NewHandler(uploadUC, listUC, getUC, downloadUC, deleteUC, storage, log, 500)
	return httpinfra.NewServer(handler, validator, port, log)
}

func TestNewServer_HealthRoute(t *testing.T) {
	srv := makeTestServer(0)
	ts := httptest.NewServer(srv.Engine())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewServer_Engine_NotNil(t *testing.T) {
	srv := makeTestServer(8099)
	assert.NotNil(t, srv.Engine())
}

func TestServer_StartAndShutdown(t *testing.T) {
	srv := makeTestServer(18765)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	assert.NoError(t, err)

	err = <-errCh
	assert.NoError(t, err)
}

package http_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos", nil)
	require.NoError(t, err)
	// No Authorization header

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, nil)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "InvalidFormatToken")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	validator := &testValidator{
		err: fmt.Errorf("token expired"),
	}
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{}, validator)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	validator := &testValidator{
		claims: &port.Claims{UserID: "user-1", UserEmail: "user@test.com"},
	}
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, validator)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer valid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuthMiddleware_BearerCaseInsensitive(t *testing.T) {
	validator := &testValidator{
		claims: &port.Claims{UserID: "user-1", UserEmail: "user@test.com"},
	}
	ts := buildServer(newTestRepo(), &testStorage{}, &testPublisher{}, &testCache{getErr: fmt.Errorf("miss")}, validator)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/videos", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "BEARER valid-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

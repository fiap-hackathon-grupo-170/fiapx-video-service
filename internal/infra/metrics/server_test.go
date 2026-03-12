package metrics_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/infra/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStartMetricsServer_HandlerRoutes(t *testing.T) {
	ctx := context.Background()
	srv := metrics.StartMetricsServer(ctx, 0, zap.NewNop())

	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()
	defer srv.Close()

	t.Run("healthz", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "ok", string(body))
	})

	t.Run("metrics", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

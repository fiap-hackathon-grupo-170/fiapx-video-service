package metrics_test

import (
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/infra/metrics"
	"github.com/stretchr/testify/assert"
)

func TestMetrics_Initialized(t *testing.T) {
	assert.NotNil(t, metrics.VideosUploadedTotal)
	assert.NotNil(t, metrics.VideosStatusUpdated)
	assert.NotNil(t, metrics.UploadDuration)
	assert.NotNil(t, metrics.ActiveUploads)
}

func TestMetrics_Counter_Increment(t *testing.T) {
	// Verify counters can be incremented without panicking
	metrics.VideosUploadedTotal.Inc()
	metrics.VideosStatusUpdated.WithLabelValues("COMPLETED").Inc()
	metrics.ActiveUploads.Inc()
	metrics.ActiveUploads.Dec()
	metrics.UploadDuration.Observe(1.5)
}

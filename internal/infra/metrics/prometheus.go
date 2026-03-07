package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	VideosUploadedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fiapx_videos_uploaded_total",
		Help: "Total number of videos uploaded",
	})

	VideosStatusUpdated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fiapx_videos_status_updated_total",
		Help: "Total number of video status updates received",
	}, []string{"status"})

	UploadDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "fiapx_video_upload_duration_seconds",
		Help:    "Duration of video upload operations",
		Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60, 120},
	})

	ActiveUploads = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fiapx_video_active_uploads",
		Help: "Number of currently active upload operations",
	})
)

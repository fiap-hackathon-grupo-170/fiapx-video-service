package config_test

import (
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/infra/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 8082, cfg.HTTPPort)
	assert.Equal(t, 8083, cfg.MetricsPort)
	assert.Equal(t, "fiapx.video", cfg.RabbitMQExchange)
	assert.Equal(t, "video.processing", cfg.RabbitMQProcessingQ)
	assert.Equal(t, "video.status", cfg.RabbitMQStatusQ)
	assert.Equal(t, "fiapx", cfg.KeycloakRealm)
	assert.Equal(t, int64(500), cfg.MaxUploadSizeMB)
	assert.Equal(t, "minioadmin", cfg.MinIOAccessKey)
	assert.Equal(t, "uploads", cfg.MinIOUploadBucket)
	assert.Equal(t, "zips", cfg.MinIOZipBucket)
	assert.False(t, cfg.MinIOUseSSL)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("METRICS_PORT", "9091")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MAX_UPLOAD_SIZE_MB", "100")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("RABBITMQ_EXCHANGE", "custom.exchange")
	t.Setenv("KEYCLOAK_REALM", "myrealm")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 9090, cfg.HTTPPort)
	assert.Equal(t, 9091, cfg.MetricsPort)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, int64(100), cfg.MaxUploadSizeMB)
	assert.True(t, cfg.MinIOUseSSL)
	assert.Equal(t, "custom.exchange", cfg.RabbitMQExchange)
	assert.Equal(t, "myrealm", cfg.KeycloakRealm)
}

func TestLoad_DatabaseURLDefault(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Contains(t, cfg.DatabaseURL, "postgres-videos")
}

func TestLoad_RedisURLDefault(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Contains(t, cfg.RedisURL, "redis://")
}

func TestLoad_JaegerEndpointDefault(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Contains(t, cfg.JaegerEndpoint, "jaeger")
}

func TestLoad_PublicURLEmpty(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.MinIOPublicURL)
}

func TestLoad_PublicURLOverride(t *testing.T) {
	t.Setenv("MINIO_PUBLIC_URL", "http://localhost:9000")
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9000", cfg.MinIOPublicURL)
}

func TestLoad_InvalidEnvType(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-number")
	_, err := config.Load()
	assert.Error(t, err)
}

package minio_test

import (
	"testing"

	minioimpl "github.com/fiapx/fiapx-video-service/internal/infra/minio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorage_ValidConfig(t *testing.T) {
	cfg := minioimpl.StorageConfig{
		Endpoint:     "localhost:9000",
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		UploadBucket: "uploads",
		ZipBucket:    "zips",
		PublicURL:    "",
	}

	storage, err := minioimpl.NewStorage(cfg)
	require.NoError(t, err)
	assert.NotNil(t, storage)
}

func TestNewStorage_WithSSL(t *testing.T) {
	cfg := minioimpl.StorageConfig{
		Endpoint:     "minio.example.com:443",
		AccessKey:    "key",
		SecretKey:    "secret",
		UseSSL:       true,
		UploadBucket: "uploads",
		ZipBucket:    "zips",
	}

	storage, err := minioimpl.NewStorage(cfg)
	require.NoError(t, err)
	assert.NotNil(t, storage)
}

func TestNewStorage_WithPublicURL(t *testing.T) {
	cfg := minioimpl.StorageConfig{
		Endpoint:     "localhost:9000",
		AccessKey:    "minioadmin",
		SecretKey:    "minioadmin",
		UseSSL:       false,
		UploadBucket: "uploads",
		ZipBucket:    "zips",
		PublicURL:    "http://public.example.com:9000/",
	}

	storage, err := minioimpl.NewStorage(cfg)
	require.NoError(t, err)
	assert.NotNil(t, storage)
}

func TestNewStorage_EmptyBuckets(t *testing.T) {
	cfg := minioimpl.StorageConfig{
		Endpoint:     "localhost:9000",
		AccessKey:    "key",
		SecretKey:    "secret",
		UploadBucket: "",
		ZipBucket:    "",
	}

	storage, err := minioimpl.NewStorage(cfg)
	require.NoError(t, err)
	assert.NotNil(t, storage)
}

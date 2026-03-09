package redis_test

import (
	"testing"

	redisinfra "github.com/fiapx/fiapx-video-service/internal/infra/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVideoCache_ValidURL(t *testing.T) {
	cache, err := redisinfra.NewVideoCache("redis://localhost:6379/0")
	require.NoError(t, err)
	assert.NotNil(t, cache)
}

func TestNewVideoCache_WithPassword(t *testing.T) {
	cache, err := redisinfra.NewVideoCache("redis://:password@localhost:6379/1")
	require.NoError(t, err)
	assert.NotNil(t, cache)
}

func TestNewVideoCache_InvalidURL(t *testing.T) {
	cache, err := redisinfra.NewVideoCache("not-a-valid-redis-url://???")
	assert.Error(t, err)
	assert.Nil(t, cache)
}

func TestNewVideoCache_RedisScheme(t *testing.T) {
	cache, err := redisinfra.NewVideoCache("redis://redis:6379/0")
	require.NoError(t, err)
	assert.NotNil(t, cache)
}

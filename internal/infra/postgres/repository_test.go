package postgres_test

import (
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/infra/postgres"
	"github.com/stretchr/testify/assert"
)

func TestNewVideoRepository_NotNil(t *testing.T) {
	repo := postgres.NewVideoRepository(nil)
	assert.NotNil(t, repo)
}

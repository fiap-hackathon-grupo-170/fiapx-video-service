package postgres_test

import (
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/infra/postgres"
	"github.com/stretchr/testify/assert"
)

func TestRunMigrations_InvalidDSN(t *testing.T) {
	err := postgres.RunMigrations("invalid-dsn", "migrations")
	assert.Error(t, err)
}

func TestRunMigrations_InvalidMigrationsPath(t *testing.T) {
	err := postgres.RunMigrations("postgresql://user:pass@localhost:5432/db?sslmode=disable", "/nonexistent/path")
	assert.Error(t, err)
}

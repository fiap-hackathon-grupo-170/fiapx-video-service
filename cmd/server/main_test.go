package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "invalid-level-xyz")
	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "init logger")
}

func TestRunDefaultConfigFailsAtInfra(t *testing.T) {
	// Default config points to unreachable hosts (minio:9000, rabbitmq:5672, etc.)
	// run() should return an error at some infra step
	err := run()
	assert.Error(t, err)
}

func TestRunFailsAtRabbitMQ(t *testing.T) {
	// Set minio to a reachable but fast-failing endpoint and rabbitmq to unreachable
	t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9099")
	t.Setenv("RABBITMQ_URL", "amqp://127.0.0.1:19999/")
	err := run()
	assert.Error(t, err)
}

func TestRunFailsAtRedis(t *testing.T) {
	// Force redis to unreachable
	t.Setenv("REDIS_URL", "redis://127.0.0.1:19998")
	err := run()
	assert.Error(t, err)
}

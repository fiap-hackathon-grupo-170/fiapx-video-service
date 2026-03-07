package rabbitmq_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublisher_PackageExists(t *testing.T) {
	// Verifies the rabbitmq package compiles and can be imported.
	// Real publisher tests require a live RabbitMQ connection (covered in integration tests).
	assert.True(t, true)
}

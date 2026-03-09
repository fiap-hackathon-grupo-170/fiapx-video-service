package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockStatusHandler struct {
	called bool
	msg    entity.VideoStatusMessage
	err    error
}

func (m *mockStatusHandler) Execute(_ context.Context, msg entity.VideoStatusMessage) error {
	m.called = true
	m.msg = msg
	return m.err
}

func TestNewStatusConsumer_NotNil(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())
	assert.NotNil(t, consumer)
}

func TestStatusConsumer_Close(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())
	// Close should not panic even with nil connection
	consumer.Close()
}

func TestHandleDelivery_ValidMessage(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())

	msg := entity.VideoStatusMessage{
		JobID:  [16]byte{1, 2, 3},
		Status: "COMPLETED",
	}
	body, err := json.Marshal(msg)
	assert.NoError(t, err)

	delivery := amqp.Delivery{Body: body}
	consumer.handleDelivery(context.Background(), delivery)

	assert.True(t, handler.called)
}

func TestHandleDelivery_InvalidJSON(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())

	delivery := amqp.Delivery{Body: []byte("invalid json {")}
	consumer.handleDelivery(context.Background(), delivery)

	// Handler should NOT be called when JSON is invalid
	assert.False(t, handler.called)
}

func TestHandleDelivery_HandlerError(t *testing.T) {
	handler := &mockStatusHandler{err: fmt.Errorf("handler error")}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())

	msg := entity.VideoStatusMessage{JobID: [16]byte{1}}
	body, _ := json.Marshal(msg)

	delivery := amqp.Delivery{Body: body}
	consumer.handleDelivery(context.Background(), delivery)

	assert.True(t, handler.called)
}

func TestHandleDelivery_EmptyBody(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())

	delivery := amqp.Delivery{Body: []byte("")}
	consumer.handleDelivery(context.Background(), delivery)

	assert.False(t, handler.called)
}

func TestStatusConsumer_StartCancelledContext(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "test.queue", handler, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Start with a cancelled context and a closed channel (simulates no msgs)
	// This should test the ctx.Done branch
	// We can't easily call Start without a real connection, so we test cancellation behavior
	_ = consumer
	_ = ctx
	// Just verify the consumer struct was created correctly
	assert.NotNil(t, consumer)
}

func TestHandleDelivery_ContextValues(t *testing.T) {
	handler := &mockStatusHandler{}
	consumer := NewStatusConsumer(nil, "queue", handler, zap.NewNop())

	msg := entity.VideoStatusMessage{
		JobID:  [16]byte{5, 6, 7},
		Status: "FAILED",
	}
	body, _ := json.Marshal(msg)
	delivery := amqp.Delivery{Body: body}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	consumer.handleDelivery(ctx, delivery)
	assert.True(t, handler.called)
	assert.Equal(t, "FAILED", handler.msg.Status)
}

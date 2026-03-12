package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// --- Mock channel for publisher ---

type mockPubChannel struct {
	publishErr      error
	exchangeDeclErr error
	closed          bool
}

func (m *mockPubChannel) PublishWithContext(_ context.Context, _, _ string, _, _ bool, _ amqp.Publishing) error {
	return m.publishErr
}

func (m *mockPubChannel) ExchangeDeclare(_, _ string, _, _, _, _ bool, _ amqp.Table) error {
	return m.exchangeDeclErr
}

func (m *mockPubChannel) Close() error {
	m.closed = true
	return nil
}

// --- Publisher tests ---

func TestPublishVideoUploaded_Success(t *testing.T) {
	mc := &mockPubChannel{}
	p := &Publisher{channel: mc, exchange: "test.exchange"}

	msg := entity.VideoUploadedMessage{
		JobID:    uuid.New(),
		UserID:   "user-1",
		VideoKey: "test.mp4",
		FileSize: 1024,
	}

	err := p.PublishVideoUploaded(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishVideoUploaded_Error(t *testing.T) {
	mc := &mockPubChannel{publishErr: fmt.Errorf("publish failed")}
	p := &Publisher{channel: mc, exchange: "test.exchange"}

	msg := entity.VideoUploadedMessage{
		JobID:    uuid.New(),
		UserID:   "user-1",
		VideoKey: "test.mp4",
	}

	err := p.PublishVideoUploaded(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Mock channel for consumer ---

type mockConsChannel struct {
	exchangeDeclErr error
	queueDeclErr    error
	queueBindErr    error
	consumeErr      error
	deliveries      chan amqp.Delivery
	closed          bool
}

func (m *mockConsChannel) ExchangeDeclare(_, _ string, _, _, _, _ bool, _ amqp.Table) error {
	return m.exchangeDeclErr
}

func (m *mockConsChannel) QueueDeclare(_ string, _, _, _, _ bool, _ amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{}, m.queueDeclErr
}

func (m *mockConsChannel) QueueBind(_, _, _ string, _ bool, _ amqp.Table) error {
	return m.queueBindErr
}

func (m *mockConsChannel) Consume(_, _ string, _, _, _, _ bool, _ amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	return m.deliveries, nil
}

func (m *mockConsChannel) Close() error {
	m.closed = true
	return nil
}

// --- Consumer tests ---

func newTestConsumer(ch *mockConsChannel) *StatusConsumer {
	return &StatusConsumer{
		openChannel: func() (consumerChannel, error) { return ch, nil },
		queue:       "test-queue",
		handler:     &mockHandler{},
		logger:      zap.NewNop(),
	}
}

type mockHandler struct {
	executeErr error
}

func (h *mockHandler) Execute(_ context.Context, _ entity.VideoStatusMessage) error {
	return h.executeErr
}

func TestConsumerStart_ChannelError(t *testing.T) {
	c := &StatusConsumer{
		openChannel: func() (consumerChannel, error) { return nil, fmt.Errorf("conn closed") },
		queue:       "test",
		handler:     &mockHandler{},
		logger:      zap.NewNop(),
	}

	err := c.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConsumerStart_ExchangeError(t *testing.T) {
	ch := &mockConsChannel{exchangeDeclErr: fmt.Errorf("exchange error")}
	c := newTestConsumer(ch)

	err := c.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConsumerStart_QueueDeclError(t *testing.T) {
	ch := &mockConsChannel{queueDeclErr: fmt.Errorf("queue error")}
	c := newTestConsumer(ch)

	err := c.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConsumerStart_QueueBindError(t *testing.T) {
	ch := &mockConsChannel{queueBindErr: fmt.Errorf("bind error")}
	c := newTestConsumer(ch)

	err := c.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConsumerStart_ConsumeError(t *testing.T) {
	ch := &mockConsChannel{consumeErr: fmt.Errorf("consume error")}
	c := newTestConsumer(ch)

	err := c.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConsumerStart_ContextCancel(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	ch := &mockConsChannel{deliveries: deliveries}
	c := newTestConsumer(ch)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- c.Start(ctx)
	}()

	cancel()
	err := <-done
	if err != nil {
		t.Fatalf("expected nil error on cancel, got %v", err)
	}
}

func TestConsumerStart_ChannelClosed(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	ch := &mockConsChannel{deliveries: deliveries}
	c := newTestConsumer(ch)

	done := make(chan error, 1)
	go func() {
		done <- c.Start(context.Background())
	}()

	close(deliveries)
	err := <-done
	if err == nil {
		t.Fatal("expected error for closed channel")
	}
}

func TestConsumerStart_ProcessMessage(t *testing.T) {
	deliveries := make(chan amqp.Delivery, 1)
	ch := &mockConsChannel{deliveries: deliveries}
	c := newTestConsumer(ch)

	msg := entity.VideoStatusMessage{
		JobID:  uuid.New(),
		UserID: "user-1",
		Status: "COMPLETED",
	}
	data, _ := json.Marshal(msg)

	deliveries <- amqp.Delivery{Body: data, Acknowledger: &mockAck{}}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- c.Start(ctx)
	}()

	// Let it process one message then cancel
	cancel()
	<-done
}

func TestConsumerClose(t *testing.T) {
	c := &StatusConsumer{}
	c.Close() // should not panic
}

// mockAck implements amqp.Acknowledger
type mockAck struct{}

func (a *mockAck) Ack(_ uint64, _ bool) error     { return nil }
func (a *mockAck) Nack(_ uint64, _, _ bool) error { return nil }
func (a *mockAck) Reject(_ uint64, _ bool) error  { return nil }

package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type StatusHandler interface {
	Execute(ctx context.Context, msg entity.VideoStatusMessage) error
}

type StatusConsumer struct {
	conn    *amqp.Connection
	queue   string
	handler StatusHandler
	logger  *zap.Logger
}

func NewStatusConsumer(conn *amqp.Connection, queue string, handler StatusHandler, logger *zap.Logger) *StatusConsumer {
	return &StatusConsumer{
		conn:    conn,
		queue:   queue,
		handler: handler,
		logger:  logger,
	}
}

func (c *StatusConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open status consumer channel: %w", err)
	}
	defer ch.Close()

	// Declare exchange and queue idempotently so the video-service can start
	// independently of the processing-service.
	if err := ch.ExchangeDeclare("fiapx.video", "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(c.queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare status queue: %w", err)
	}
	if err := ch.QueueBind(c.queue, "video.status", "fiapx.video", false, nil); err != nil {
		return fmt.Errorf("bind status queue: %w", err)
	}

	msgs, err := ch.Consume(
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume status queue: %w", err)
	}

	c.logger.Info("status consumer started", zap.String("queue", c.queue))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("status consumer stopping")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("status queue channel closed")
			}
			c.handleDelivery(ctx, d)
		}
	}
}

func (c *StatusConsumer) Close() {}

func (c *StatusConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) {
	var msg entity.VideoStatusMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		c.logger.Error("failed to unmarshal status message",
			zap.Error(err),
			zap.ByteString("body", d.Body),
		)
		_ = d.Nack(false, false)
		return
	}

	if err := c.handler.Execute(ctx, msg); err != nil {
		c.logger.Error("failed to handle status update",
			zap.Error(err),
			zap.String("job_id", msg.JobID.String()),
		)
		_ = d.Nack(false, false)
		return
	}

	_ = d.Ack(false)
}

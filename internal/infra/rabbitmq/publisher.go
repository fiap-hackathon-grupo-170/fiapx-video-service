package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fiapx/fiapx-video-service/internal/domain/entity"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(conn *amqp.Connection, exchange string) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open publisher channel: %w", err)
	}

	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		return nil, fmt.Errorf("declare exchange %s: %w", exchange, err)
	}

	return &Publisher{channel: ch, exchange: exchange}, nil
}

func (p *Publisher) PublishVideoUploaded(ctx context.Context, msg entity.VideoUploadedMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal video uploaded message: %w", err)
	}

	return p.channel.PublishWithContext(ctx,
		p.exchange,
		"video.processing",
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
		},
	)
}

package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/D1sordxr/packages/rabbitmq"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher implements ports.Publisher on top of the notification topology.
type Publisher struct {
	p *rabbitmq.Publisher
}

func NewPublisher(p *rabbitmq.Publisher) *Publisher {
	return &Publisher{p: p}
}

func (p *Publisher) Publish(ctx context.Context, id uuid.UUID, delay time.Duration) error {
	const op = "broker.rabbitmq.Publisher.Publish"

	msg, err := message(id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	exchange := NotificationsExchange
	if delay > 0 {
		exchange = WaitExchange
		msg.Expiration = rabbitmq.Expiration(delay)
	}

	if err = p.p.Publish(ctx, exchange, routingKey, msg); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (p *Publisher) PublishRetry(ctx context.Context, id uuid.UUID) error {
	const op = "broker.rabbitmq.Publisher.PublishRetry"

	msg, err := message(id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err = p.p.Publish(ctx, RetryExchange, routingKey, msg); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func message(id uuid.UUID) (amqp.Publishing, error) {
	body, err := EncodeMessage(id)
	if err != nil {
		return amqp.Publishing{}, err
	}

	return amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    id.String(),
		Timestamp:    time.Now(),
		Body:         body,
	}, nil
}

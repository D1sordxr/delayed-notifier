package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/pkg/logger"

	"github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	log appPorts.Logger
	q   *Queue
}

func NewConsumer(log appPorts.Logger, q *Queue) *Consumer {
	return &Consumer{
		log: log,
		q:   q,
	}
}

func (c *Consumer) StartConsuming(
	ctx context.Context,
	handler func(ctx context.Context, m *model.Notification) error,
) error {
	const op = "broker.rabbitmq.Consumer.StartConsuming"

	deliveries, err := c.q.consume()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	c.log.Info("Starting RabbitMQ consumer")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Consumer stopped by context")
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				c.log.Info("Deliveries channel closed")
				return nil
			}
			c.processDelivery(ctx, delivery, handler)
		}
	}
}

func (c *Consumer) processDelivery(
	ctx context.Context,
	delivery amqp091.Delivery,
	handler func(ctx context.Context, m *model.Notification) error,
) {
	const op = "broker.rabbitmq.Consumer.processDelivery"
	withFields := logger.WithFields("op", op)

	var notification model.Notification
	if err := json.Unmarshal(delivery.Body, &notification); err != nil {
		c.log.Error("Failed to unmarshal notification",
			withFields("error", err.Error(), "message_id", delivery.MessageId)...,
		)

		if err = delivery.Reject(false); err != nil {
			c.log.Error("Failed to reject invalid message", withFields("error", err.Error())...)
		}
		return
	}

	c.log.Debug("Processing notification",
		withFields("notification_id", notification.ID, "message_id", delivery.MessageId)...,
	)

	if err := handler(ctx, &notification); err != nil {
		c.log.Error("Handler failed",
			withFields("error", err.Error(), "notification_id", notification.ID)...,
		)

		if err = delivery.Nack(false, true); err != nil {
			c.log.Error("Failed to nack message", withFields("error", err.Error())...)
		}
		return
	}

	if err := delivery.Ack(false); err != nil {
		c.log.Error("Failed to ack message",
			withFields("error", err.Error(), "notification_id", notification.ID)...,
		)
		return
	}

	c.log.Debug("Notification processed successfully",
		withFields("notification_id", notification.ID, "message_id", delivery.MessageId)...,
	)
}

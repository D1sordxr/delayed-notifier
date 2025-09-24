package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
)

type Sender struct {
	log      appPorts.Logger
	conn     *rabbitmq.Connection
	channel  *rabbitmq.Channel
	exchange *rabbitmq.Exchange
}

func NewSender(
	log appPorts.Logger,
	conn *rabbitmq.Connection,
) (*Sender, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	exchange := rabbitmq.NewExchange("delayed_notifications", "topic")
	exchange.Durable = true
	exchange.Args = amqp091.Table{
		"x-delayed-type": "direct",
	}

	if err = exchange.BindToChannel(channel); err != nil {
		return nil, fmt.Errorf("failed to declare delayed exchange: %w", err)
	}

	queueManager := rabbitmq.NewQueueManager(channel)

	queueCfg := rabbitmq.QueueConfig{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
	}

	_, err = queueManager.DeclareQueue("notifications_queue", queueCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = channel.QueueBind(
		"notifications_queue",
		"notifications",
		"delayed_notifications",
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &Sender{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
		log:      log,
	}, nil
}

func (s *Sender) PublishDelayed(notification model.Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	delay := time.Until(notification.Scheduled)
	if delay < 0 {
		delay = 256
	}

	publisher := rabbitmq.NewPublisher(s.channel, s.exchange.Name())

	pubOptions := rabbitmq.PublishingOptions{
		Headers: amqp091.Table{
			"x-delay": int(delay.Milliseconds()),
		},
	}

	err = publisher.Publish(
		body,
		"notifications",
		"application/json",
		pubOptions,
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	s.log.Info("Notification published to RabbitMQ with delay",
		"notification_id", notification.ID.String(),
		"delay_ms", delay.Milliseconds(),
		"scheduled_at", notification.ScheduledAt.Format(time.RFC3339),
	)

	return nil
}

func (s *Sender) Close() error {
	if s.channel != nil {
		if err := s.channel.Close(); err != nil {
			s.log.Error("Failed to close channel", "error", err.Error())
		}
	}

	return nil
}

func (s *Sender) HealthCheck() error {
	if s.conn == nil || s.conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection is closed")
	}
	return nil
}

func (s *Sender) Run(ctx context.Context) error {
	const op = "rabbitmq.notifications.Sender.Run"

	s.log.Info("Starting RabbitMQ notification sender", "operation", op)

	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-healthTicker.C:
			if err := s.HealthCheck(); err != nil {
				s.log.Error(
					"Failed to check RabbitMQ notification sender health",
					"error", err.Error(),
				)
				return fmt.Errorf("%s: failed to check RabbitMQ connection: %w", op, err)
			}
		}
	}
}

func (s *Sender) Shutdown(_ context.Context) error {
	return s.conn.Close()
}

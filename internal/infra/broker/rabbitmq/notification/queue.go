package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/vo"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
)

type Queue struct {
	log  appPorts.Logger
	conn *rabbitmq.Connection
	ch   *rabbitmq.Channel
}

func NewQueue(
	log appPorts.Logger,
	conn *rabbitmq.Connection,
	declareExchange bool,
) (*Queue, error) {
	const op = "broker.rabbitmq.NewQueue"

	channel, err := conn.Channel()
	if err != nil {
		log.Error("Failed to open a channel", "error", err.Error())
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	queue := Queue{
		log:  log,
		conn: conn,
		ch:   channel,
	}

	if err = queue.setup(declareExchange); err != nil {
		log.Error("Failed to setup queue", "error", err.Error())
		return nil, fmt.Errorf("%s: setup: %w", op, err)
	}

	return &queue, nil
}

func (q *Queue) setup(declareExchange bool) error {
	exchangesToDeclare := []struct {
		name string
		kind string
	}{
		{vo.NotificationsExchange, vo.Direct},
		{vo.WaitExchange, vo.Direct},
		{vo.RetryExchange, vo.Direct},
	}

	if declareExchange {
		for _, exchange := range exchangesToDeclare {
			if err := q.ch.ExchangeDeclare(
				exchange.name,
				exchange.kind,
				true,
				false,
				false,
				false,
				nil,
			); err != nil {
				return fmt.Errorf("failed to declare exchange %s: %w", exchange.name, err)
			}
		}
	}

	if _, err := q.ch.QueueDeclare(
		vo.NotificationsQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", vo.NotificationsQueue, err)
	}
	waitQueueArgs := amqp091.Table{"x-dead-letter-exchange": vo.NotificationsExchange}
	if _, err := q.ch.QueueDeclare(
		vo.WaitQueue,
		true,
		false,
		false,
		false,
		waitQueueArgs,
	); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", vo.WaitQueue, err)
	}
	retryQueueArgs := amqp091.Table{"x-dead-letter-exchange": vo.NotificationsExchange}
	if _, err := q.ch.QueueDeclare(
		vo.RetryQueue,
		true,
		false,
		false,
		false,
		retryQueueArgs,
	); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", vo.RetryQueue, err)
	}

	if err := q.ch.QueueBind(
		vo.NotificationsQueue,
		"",
		vo.NotificationsExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf(
			"failed to bind queue %s to exchange %s: %w",
			vo.NotificationsQueue,
			vo.NotificationsExchange,
			err,
		)
	}
	if err := q.ch.QueueBind(
		vo.WaitQueue,
		"",
		vo.WaitExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf(
			"failed to bind queue %s to exchange %s: %w",
			vo.WaitQueue,
			vo.WaitExchange,
			err,
		)
	}
	if err := q.ch.QueueBind(
		vo.RetryQueue,
		"",
		vo.RetryExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf(
			"failed to bind queue %s to exchange %s: %w",
			vo.RetryQueue,
			vo.RetryExchange,
			err,
		)
	}

	return nil
}

func (q *Queue) publish(ctx context.Context, n *model.Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	delay := time.Until(n.ScheduledAt)
	if delay < 0 {
		delay = 10 * time.Millisecond
	}

	msg := amqp091.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp091.Persistent,
		Expiration:   strconv.FormatInt(delay.Milliseconds(), 10),
	}

	return q.ch.PublishWithContext(ctx, vo.NotificationsExchange, "", false, false, msg)
}

func (q *Queue) publishRetry(ctx context.Context, n *model.Notification, retryDelay time.Duration) error {
	body, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal notification for retry: %w", err)
	}

	msg := amqp091.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp091.Persistent,
		Expiration:   fmt.Sprintf("%d", retryDelay.Milliseconds()),
	}

	return q.ch.PublishWithContext(ctx, vo.RetryExchange, "", false, false, msg)
}

func (q *Queue) Close() error {
	if q.ch != nil {
		return q.ch.Close()
	}
	return nil
}

func (q *Queue) consume() (<-chan amqp091.Delivery, error) {
	if err := q.ch.Qos(
		1,
		0,
		false,
	); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return q.ch.Consume(
		vo.NotificationsQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

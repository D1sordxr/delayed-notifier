package rabbitmq

import (
	"context"
	"fmt"
	"time"
	appPorts "wb-tech-l3/internal/domain/app/ports"

	"github.com/wb-go/wbf/rabbitmq"
)

type Reader struct {
	log      appPorts.Logger
	conn     *rabbitmq.Connection
	channel  *rabbitmq.Channel
	consumer *rabbitmq.Consumer
	msgChan  chan []byte
}

func NewReader(
	log appPorts.Logger,
	conn *rabbitmq.Connection,
) (*Reader, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
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

	consumerConfig := rabbitmq.NewConsumerConfig("notifications_queue")
	consumerConfig.AutoAck = false

	consumer := rabbitmq.NewConsumer(channel, consumerConfig)

	return &Reader{
		log:      log,
		conn:     conn,
		channel:  channel,
		consumer: consumer,
		msgChan:  make(chan []byte, chanBufferSize),
	}, nil
}

func (r *Reader) GetMessageChan() <-chan []byte {
	return r.msgChan
}

func (r *Reader) Close() error {
	defer close(r.msgChan)
	if r.channel != nil || !r.channel.IsClosed() {
		if err := r.channel.Close(); err != nil {
			r.log.Error("Failed to close channel", "error", err.Error())
			return err
		}
	}

	return nil
}

func (r *Reader) HealthCheck() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection is closed")
	}
	return nil
}

func (r *Reader) Run(ctx context.Context) error {
	const op = "rabbitmq.notifications.MessagePipe.Run"

	r.log.Info("Starting RabbitMQ notification reader...")

	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()

	go func() {
		if err := r.consumer.Consume(r.msgChan); err != nil {
			r.log.Error("Failed to start consumer", "error", err.Error())
			return
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-healthTicker.C:
			if err := r.HealthCheck(); err != nil {
				r.log.Error(
					"Failed to check RabbitMQ notification reader health",
					"error", err.Error(),
				)
				return fmt.Errorf("%s: failed to check RabbitMQ connection: %w", op, err)
			}
		}
	}
}

func (r *Reader) Shutdown(_ context.Context) error {
	return r.Close()
}

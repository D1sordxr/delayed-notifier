package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"wb-tech-l3/internal/domain/core/notification/model"

	"github.com/wb-go/wbf/rabbitmq"

	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/vo"
)

const (
	chanBufferSize  = 1024
	rebootTime      = 3 * time.Second
	publishTimeout  = 5 * time.Second
	shutdownTimeout = 10 * time.Second
)

type Adapter struct {
	log            appPorts.Logger
	channel        *rabbitmq.Channel
	exchange       *rabbitmq.Exchange
	consumer       *rabbitmq.Consumer
	publisher      *rabbitmq.Publisher
	msgChan        chan *model.Notification
	rawMsgChan     chan []byte
	publishTimeout time.Duration
	mu             sync.RWMutex
	isRunning      bool
}

func NewNotificationAdapter(
	log appPorts.Logger,
	conn *rabbitmq.Connection,
) (*Adapter, error) {
	channel, err := conn.Channel()
	if err != nil {
		log.Error("Failed to open channel", "error", err.Error())
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	exchange := rabbitmq.NewExchange(
		vo.BrokerExchangeNotification,
		"topic",
	)

	if err = exchange.BindToChannel(channel); err != nil {
		log.Error("Failed to declare exchange", "error", err.Error())
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	queueManager := rabbitmq.NewQueueManager(channel)

	queueConfig := rabbitmq.QueueConfig{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
	}

	queue, err := queueManager.DeclareQueue(vo.NotificationQueue, queueConfig)
	if err != nil {
		log.Error("Failed to declare queue", "error", err.Error())
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = channel.QueueBind(
		queue.Name,
		"notifications",
		exchange.Name(),
		false,
		nil,
	)
	if err != nil {
		log.Error("Failed to bind queue", "error", err.Error())
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	consumerConfig := rabbitmq.NewConsumerConfig(vo.NotificationQueue)
	consumer := rabbitmq.NewConsumer(channel, consumerConfig)

	publisher := rabbitmq.NewPublisher(channel, exchange.Name())

	return &Adapter{
		log:            log,
		channel:        channel,
		exchange:       exchange,
		consumer:       consumer,
		publisher:      publisher,
		msgChan:        make(chan *model.Notification, chanBufferSize),
		rawMsgChan:     make(chan []byte, chanBufferSize),
		publishTimeout: publishTimeout,
		isRunning:      false,
	}, nil
}

func (a *Adapter) GetMessageChan() <-chan *model.Notification {
	return a.msgChan
}

func (a *Adapter) Publish(ctx context.Context, n *model.Notification) error {
	a.log.Debug("Publishing notification", "id", n.ID)

	body, err := json.Marshal(n)
	if err != nil {
		a.log.Error("Failed to marshal notification", "error", err.Error(), "id", n.ID)
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	publishCtx, cancel := context.WithTimeout(ctx, a.publishTimeout)
	defer cancel()

	done := make(chan error, 1)

	go func() {
		if err = a.publisher.Publish(
			body,
			"notifications",
			"application/json",
			rabbitmq.PublishingOptions{},
		); err != nil {
			done <- fmt.Errorf("failed to publish notification: %w", err)
			return
		}
		done <- nil
	}()

	select {
	case <-publishCtx.Done():
		a.log.Error("Publish timeout exceeded", "id", n.ID, "timeout", a.publishTimeout)
		return fmt.Errorf("publish timeout exceeded: %w", publishCtx.Err())
	case err = <-done:
		if err != nil {
			a.log.Error("Failed to publish notification", "error", err.Error(), "id", n.ID)
			return err
		}
		a.log.Debug("Notification published successfully", "id", n.ID)
		return nil
	}
}

func (a *Adapter) HealthCheck() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return fmt.Errorf("adapter is not running")
	}

	if a.channel.IsClosed() {
		return fmt.Errorf("rabbitmq channel is closed")
	}

	return nil
}

func (a *Adapter) Run(ctx context.Context) error {
	const op = "rabbitmq.notification.Adapter.Run"

	a.mu.Lock()
	if a.isRunning {
		a.mu.Unlock()
		return fmt.Errorf("%s: adapter is already running", op)
	}
	a.isRunning = true
	a.mu.Unlock()

	a.log.Info("Starting notification RabbitMQ adapter...")

	defer func() {
		a.mu.Lock()
		a.isRunning = false
		a.mu.Unlock()
		a.log.Info("Notification RabbitMQ adapter stopped")
	}()

	goroutineCtx, cancelGoroutines := context.WithCancel(ctx)
	defer cancelGoroutines()

	var wg sync.WaitGroup

	wg.Add(1)
	go a.runConsumer(goroutineCtx, &wg)

	wg.Add(1)
	go a.runMessageProcessor(goroutineCtx, &wg)

	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()

	a.log.Info("Notification RabbitMQ adapter started successfully")

	for {
		select {
		case <-ctx.Done():
			a.log.Info("Shutdown signal received, stopping adapter...")
			cancelGoroutines()

			waitDone := make(chan struct{})
			go func() {
				wg.Wait()
				close(waitDone)
			}()

			select {
			case <-waitDone:
				a.log.Info("All goroutines stopped gracefully")
			case <-time.After(shutdownTimeout):
				a.log.Warn("Timeout waiting for goroutines to stop")
			}
			return nil

		case <-healthTicker.C:
			if err := a.HealthCheck(); err != nil {
				a.log.Error("Health check failed", "error", err.Error())
				cancelGoroutines()
				wg.Wait()
				return fmt.Errorf("%s: health check failed: %w", op, err)
			} else {
				a.log.Debug("Health check passed")
			}
		}
	}
}

func (a *Adapter) runConsumer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("Recovered from panic in consumer", "panic", r)
		}
	}()

	a.log.Info("Starting RabbitMQ consumer...")

	for {
		select {
		case <-ctx.Done():
			a.log.Info("Stopping RabbitMQ consumer...")
			return
		default:
			if a.channel.IsClosed() {
				a.log.Warn("Channel is closed, waiting for reconnect...")
				time.Sleep(rebootTime)
				continue
			}

			a.log.Debug("Starting message consumption...")
			if err := a.consumer.Consume(a.rawMsgChan); err != nil {
				if !a.channel.IsClosed() {
					a.log.Error("Consumer failed, restarting...", "error", err.Error())
				} else {
					a.log.Warn("Consumer stopped due to closed channel")
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(rebootTime):
					continue
				}
			}
		}
	}
}

func (a *Adapter) runMessageProcessor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("Recovered from panic in message processor", "panic", r)
		}
	}()

	a.log.Info("Starting message processor...")

	for {
		select {
		case <-ctx.Done():
			a.log.Info("Stopping message processor...")
			return
		case rawMsg, ok := <-a.rawMsgChan:
			if !ok {
				a.log.Warn("Raw message channel closed")
				return
			}

			a.processMessage(rawMsg)
		}
	}
}

func (a *Adapter) processMessage(rawMsg []byte) {
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("Recovered from panic while processing message", "panic", r)
		}
	}()

	var notification model.Notification
	if err := json.Unmarshal(rawMsg, &notification); err != nil {
		a.log.Error("Failed to unmarshal raw message", "error", err.Error())
		return
	}

	a.log.Debug("Processing notification", "id", notification.ID)

	select {
	case a.msgChan <- &notification:
		a.log.Debug("Notification sent to worker channel", "id", notification.ID)
	case <-time.After(1 * time.Second):
		a.log.Warn("Timeout sending notification to worker channel", "id", notification.ID)
	}
}

func (a *Adapter) Shutdown(ctx context.Context) error {
	const op = "rabbitmq.notification.Adapter.Shutdown"

	a.log.Info("Shutting down RabbitMQ adapter...")

	done := make(chan struct{})
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	go func() {
		defer close(done)
		a.mu.Lock()
		a.isRunning = false
		a.mu.Unlock()

		a.safeCloseChannels()

		if err := a.channel.Close(); err != nil {
			a.log.Error("Failed to close channel", "error", err.Error())
			return
		}
	}()

	select {
	case <-shutdownCtx.Done():
		a.log.Warn("RabbitMQ adapter shutdown timed out")
		return shutdownCtx.Err()
	case <-done:
		return nil
	}
}

func (a *Adapter) safeCloseChannels() {
	defer func() {
		if r := recover(); r != nil {
			a.log.Warn("Recovered from panic while closing channels", "panic", r)
		}
	}()

	select {
	case _, ok := <-a.msgChan:
		if ok {
			close(a.msgChan)
			a.log.Debug("Message channel closed")
		}
	default:
		close(a.msgChan)
		a.log.Debug("Message channel closed")
	}

	select {
	case _, ok := <-a.rawMsgChan:
		if ok {
			close(a.rawMsgChan)
			a.log.Debug("Raw message channel closed")
		}
	default:
		close(a.rawMsgChan)
		a.log.Debug("Raw message channel closed")
	}
}

func (a *Adapter) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isRunning
}

func (a *Adapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"is_running":       a.isRunning,
		"channel_closed":   a.channel.IsClosed(),
		"msg_chan_len":     len(a.msgChan),
		"msg_chan_cap":     cap(a.msgChan),
		"raw_msg_chan_len": len(a.rawMsgChan),
		"raw_msg_chan_cap": cap(a.rawMsgChan),
	}
}

package rabbitmq

import (
	"time"

	"github.com/D1sordxr/packages/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	NotificationsQueue = "notifications.queue.process"
	WaitQueue          = "wait.queue.delay"
	RetryQueue         = "retry.queue.delay"

	NotificationsExchange = "notifications.exchange"
	WaitExchange          = "wait.exchange"
	RetryExchange         = "retry.exchange"

	routingKey = "notification"
)

// Topology declares the notification flow:
//
//	wait.exchange  -> wait.queue.delay  (per-message TTL)  --dead-letter--> notifications.exchange
//	retry.exchange -> retry.queue.delay (queue TTL = retryDelay) --dead-letter--> notifications.exchange
//	notifications.exchange -> notifications.queue.process -> dispatcher
//
// The wait queue holds notifications claimed shortly before they are due, so
// they reach the dispatcher on time rather than on the next poll. RabbitMQ
// expires messages only at the head of a queue; the scheduler keeps TTLs in
// the wait queue within its short look-ahead window, and the retry queue uses
// one queue-level TTL, so messages never wait behind much longer ones.
func Topology(retryDelay time.Duration) rabbitmq.Topology {
	return rabbitmq.Topology{
		Exchanges: []rabbitmq.Exchange{
			{Name: NotificationsExchange, Kind: amqp.ExchangeDirect, Durable: true},
			{Name: WaitExchange, Kind: amqp.ExchangeDirect, Durable: true},
			{Name: RetryExchange, Kind: amqp.ExchangeDirect, Durable: true},
		},
		Queues: []rabbitmq.Queue{
			{Name: NotificationsQueue, Durable: true},
			{Name: WaitQueue, Durable: true, Args: rabbitmq.DelayQueueArgs(NotificationsExchange, "", 0)},
			{Name: RetryQueue, Durable: true, Args: rabbitmq.DelayQueueArgs(NotificationsExchange, "", retryDelay)},
		},
		Bindings: []rabbitmq.Binding{
			{Queue: NotificationsQueue, Exchange: NotificationsExchange, RoutingKey: routingKey},
			{Queue: WaitQueue, Exchange: WaitExchange, RoutingKey: routingKey},
			{Queue: RetryQueue, Exchange: RetryExchange, RoutingKey: routingKey},
		},
	}
}

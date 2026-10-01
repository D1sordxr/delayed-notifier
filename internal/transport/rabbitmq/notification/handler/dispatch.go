package handler

import (
	"context"
	"fmt"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/port"
	broker "github.com/D1sordxr/delayed-notifier/internal/infra/broker/rabbitmq/notification"

	"github.com/D1sordxr/packages/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// NewDispatchHandler returns the consumer handler of the notifications queue.
// A returned error rejects the delivery; the notification stays pending in
// the database and is re-claimed by the scheduler once it is stale.
func NewDispatchHandler(uc port.DispatcherUseCase) rabbitmq.Handler {
	return func(ctx context.Context, d amqp.Delivery) error {
		msg, err := broker.DecodeMessage(d.Body)
		if err != nil {
			return fmt.Errorf("transport.rabbitmq.dispatch: %w", err)
		}

		return uc.Dispatch(ctx, msg.ID)
	}
}

package rabbitmq

import (
	"context"
	"time"
	"wb-tech-l3/internal/domain/core/notification/model"
)

type Producer struct {
	q *Queue
}

func NewProducer(q *Queue) *Producer {
	return &Producer{
		q: q,
	}
}

func (p *Producer) Publish(ctx context.Context, n *model.Notification) error {
	return p.q.publish(ctx, n)
}

func (p *Producer) PublishRetry(ctx context.Context, n *model.Notification, retryDelay time.Duration) error {
	return p.q.publishRetry(ctx, n, retryDelay)
}

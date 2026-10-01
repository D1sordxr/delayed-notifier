package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Publisher hands notifications over to the dispatcher through the broker.
type Publisher interface {
	// Publish delivers the notification to the dispatcher after delay
	// (immediately when delay <= 0).
	Publish(ctx context.Context, id uuid.UUID, delay time.Duration) error
	// PublishRetry delivers the notification to the dispatcher after the retry delay.
	PublishRetry(ctx context.Context, id uuid.UUID) error
}

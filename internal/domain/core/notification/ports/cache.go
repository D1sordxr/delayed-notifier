package ports

import (
	"context"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"

	"github.com/google/uuid"
)

type CacheStore interface {
	// Get returns nil, nil on a cache miss.
	Get(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	// Set stores the notification unless a newer state is already cached.
	Set(ctx context.Context, n *model.Notification) error
}

// CacheWriter updates the cache off the request path. It is best effort:
// writes may be dropped under load.
type CacheWriter interface {
	Enqueue(n *model.Notification)
}

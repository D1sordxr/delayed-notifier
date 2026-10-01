package ports

import (
	"context"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
)

// Sender delivers a notification to its recipient over its channel.
type Sender interface {
	Send(ctx context.Context, n *model.Notification) error
}

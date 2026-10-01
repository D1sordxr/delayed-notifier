// Package sender delivers notifications over their channels.
package sender

import (
	"context"
	"errors"
	"fmt"

	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
)

var ErrUnsupportedChannel = errors.New("unsupported channel")

// Router sends a notification with the sender registered for its channel.
type Router map[vo.Channel]ports.Sender

func (r Router) Send(ctx context.Context, n *model.Notification) error {
	s, ok := r[n.Channel]
	if !ok {
		return fmt.Errorf("sender.Router: %w: %s", ErrUnsupportedChannel, n.Channel)
	}
	return s.Send(ctx, n)
}

// Log "delivers" notifications by logging them. It stands in for the real
// email (SMTP), Telegram (Bot API) and SMS gateway integrations, which plug
// in as other ports.Sender implementations.
type Log struct {
	log appPorts.Logger
}

func NewLog(log appPorts.Logger) *Log {
	return &Log{log: log}
}

func (s *Log) Send(_ context.Context, n *model.Notification) error {
	s.log.Info("Delivering notification",
		"notification_id", n.ID.String(),
		"channel", n.Channel.String(),
		"recipient", recipient(n),
		"subject", n.Subject,
	)
	return nil
}

func recipient(n *model.Notification) any {
	switch {
	case n.EmailTo != nil:
		return *n.EmailTo
	case n.TelegramChatID != nil:
		return *n.TelegramChatID
	case n.SmsTo != nil:
		return *n.SmsTo
	default:
		return nil
	}
}

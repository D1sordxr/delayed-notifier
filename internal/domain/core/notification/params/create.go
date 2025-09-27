package params

import (
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
)

type CreateNotificationParams struct {
	Subject        string     `json:"subject"`
	Message        string     `json:"message"`
	AuthorID       *string    `json:"author_id"`
	EmailTo        *string    `json:"email_to"`
	TelegramChatID *int64     `json:"telegram_chat_id"`
	SmsTo          *string    `json:"sms_to"`
	Channel        vo.Channel `json:"channel"`
	Status         vo.Status  `json:"status"`
	Attempts       int16      `json:"attempts"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
}

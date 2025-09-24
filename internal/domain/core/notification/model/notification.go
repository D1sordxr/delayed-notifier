package model

import (
	"time"
	"wb-tech-l3/internal/domain/core/notification/vo"

	"github.com/google/uuid"
)

type Notification struct {
	ID       uuid.UUID `json:"id"`
	Subject  string    `json:"subject"`
	Message  string    `json:"message"`
	AuthorID *string   `json:"author_id"` // Может быть NULL

	// Поля получателя (в зависимости от канала)
	EmailTo        *string `json:"email_to"`
	TelegramChatID *int64  `json:"telegram_chat_id"`
	SmsTo          *string `json:"sms_to"`

	// Основные поля уведомления
	Channel  vo.Channel `json:"channel"`  // email, telegram, sms
	Status   vo.Status  `json:"status"`   // pending, sent, failed, declined
	Attempts int16      `json:"attempts"` // Количество попыток отправки

	// Временные метки
	ScheduledAt time.Time  `json:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at"` // Может быть NULL
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

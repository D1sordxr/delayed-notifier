package input

import "time"

type CreateNotifyInput struct {
	AuthorID   string    `json:"user_id"`
	Subject    string    `json:"subject"`
	Message    string    `json:"message"`
	Channel    string    `json:"channel"`
	EmailTo    *string   `json:"email_to,omitempty"`
	TelegramID *int64    `json:"telegram_id,omitempty"`
	SmsTo      *string   `json:"sms_to,omitempty"`
	Scheduled  time.Time `json:"scheduled_at"`
}

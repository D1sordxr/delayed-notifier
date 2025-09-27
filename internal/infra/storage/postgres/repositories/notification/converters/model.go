package converters

import (
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification/gen"
)

func ConvertGenToDomain(rawModel *gen.Notification) *model.Notification {
	channel, _ := vo.ParseChannel(string(rawModel.Channel))
	status, _ := vo.ParseStatus(string(rawModel.Status))

	return &model.Notification{
		ID:             rawModel.ID,
		Subject:        rawModel.Subject,
		Message:        rawModel.Message,
		AuthorID:       &rawModel.AuthorID.String,
		EmailTo:        &rawModel.EmailTo.String,
		TelegramChatID: &rawModel.TelegramChatID.Int64,
		SmsTo:          &rawModel.SmsTo.String,
		Channel:        channel,
		Status:         status,
		Attempts:       rawModel.Attempts,
		ScheduledAt:    rawModel.ScheduledAt,
		SentAt:         &rawModel.SentAt.Time,
		CreatedAt:      rawModel.CreatedAt,
		UpdatedAt:      rawModel.UpdatedAt,
	}
}

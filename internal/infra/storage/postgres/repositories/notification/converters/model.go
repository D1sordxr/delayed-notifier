package converters

import (
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/vo"
	"wb-tech-l3/internal/infra/storage/postgres/repositories/notification/gen"
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

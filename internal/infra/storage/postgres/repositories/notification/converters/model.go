package converters

import (
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification/gen"
)

func ConvertGenToDomain(rawModel *gen.Notification) *model.Notification {
	// The database enums match the domain values, so parsing cannot fail.
	channel, _ := vo.ParseChannel(string(rawModel.Channel))
	status, _ := vo.ParseStatus(string(rawModel.Status))

	return &model.Notification{
		ID:             rawModel.ID,
		Subject:        rawModel.Subject,
		Message:        rawModel.Message,
		AuthorID:       rawModel.AuthorID,
		EmailTo:        rawModel.EmailTo,
		TelegramChatID: rawModel.TelegramChatID,
		SmsTo:          rawModel.SmsTo,
		Channel:        channel,
		Status:         status,
		Attempts:       rawModel.Attempts,
		ScheduledAt:    rawModel.ScheduledAt,
		SentAt:         rawModel.SentAt,
		CreatedAt:      rawModel.CreatedAt,
		UpdatedAt:      rawModel.UpdatedAt,
	}
}

func ConvertGenSliceToDomain(rawModels []gen.Notification) []*model.Notification {
	notifications := make([]*model.Notification, len(rawModels))
	for i := range rawModels {
		notifications[i] = ConvertGenToDomain(&rawModels[i])
	}
	return notifications
}

func ConvertCreateParams(p params.CreateNotificationParams) gen.CreateNotificationParams {
	return gen.CreateNotificationParams{
		Subject:        p.Subject,
		Message:        p.Message,
		AuthorID:       p.AuthorID,
		EmailTo:        p.EmailTo,
		TelegramChatID: p.TelegramChatID,
		SmsTo:          p.SmsTo,
		Channel:        gen.ChannelType(p.Channel.String()),
		Status:         gen.NotificationStatus(p.Status.String()),
		Attempts:       p.Attempts,
		ScheduledAt:    p.ScheduledAt,
	}
}

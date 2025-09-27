package converters

import (
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification/gen"
	"github.com/D1sordxr/delayed-notifier/pkg/pgutil"
)

func ConvertCreateParams(p params.CreateNotificationParams) gen.CreateNotificationParams {
	return gen.CreateNotificationParams{
		Subject:        p.Subject,
		Message:        p.Message,
		AuthorID:       pgutil.ToNullString(p.AuthorID),
		EmailTo:        pgutil.ToNullString(p.EmailTo),
		TelegramChatID: pgutil.ToNullInt64(p.TelegramChatID),
		SmsTo:          pgutil.ToNullString(p.SmsTo),
		Channel:        gen.ChannelType(p.Channel.String()),
		Status:         gen.NotificationStatus(p.Status.String()),
		Attempts:       p.Attempts,
		ScheduledAt:    p.ScheduledAt,
	}
}

func ConvertUpdateParams(p params.UpdateNotificationStatusParams) gen.UpdateNotificationStatusParams {
	return gen.UpdateNotificationStatusParams{
		ID:       p.ID,
		Status:   gen.NotificationStatus(p.Status.String()),
		Attempts: p.Attempts,
		SentAt:   pgutil.ToNullTime(p.SentAt),
	}
}

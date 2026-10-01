package usecase

import (
	"context"
	"fmt"
	"net/mail"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/input"
	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/pkg/logger"

	"github.com/google/uuid"
)

type UseCase struct {
	log   appPorts.Logger
	cache ports.CacheStore
	cw    ports.CacheWriter
	repo  ports.Repository
}

func NewUseCase(
	log appPorts.Logger,
	cache ports.CacheStore,
	cw ports.CacheWriter,
	repo ports.Repository,
) *UseCase {
	return &UseCase{
		log:   log,
		cache: cache,
		cw:    cw,
		repo:  repo,
	}
}

func (uc *UseCase) Create(ctx context.Context, input input.CreateNotifyInput) (*model.Notification, error) {
	const op = "notification.UseCase.Create"
	logFields := logger.WithFields("operation", op)

	// Validation errors are shown to the client as is, without the operation prefix.
	channel, err := vo.ParseChannel(input.Channel)
	if err != nil {
		return nil, fmt.Errorf("%w %q", err, input.Channel)
	}

	if err = validateRecipient(input, channel); err != nil {
		return nil, fmt.Errorf("%w: %w", errorx.ErrInvalidRecipient, err)
	}

	notification, err := uc.repo.Create(ctx, params.CreateNotificationParams{
		Subject:        input.Subject,
		Message:        input.Message,
		AuthorID:       &input.AuthorID,
		EmailTo:        input.EmailTo,
		TelegramChatID: input.TelegramID,
		SmsTo:          input.SmsTo,
		Channel:        channel,
		Status:         vo.Pending,
		Attempts:       0,
		ScheduledAt:    input.Scheduled,
	})
	if err != nil {
		uc.log.Error("Error saving notification to database", logFields("error", err.Error())...)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	uc.cw.Enqueue(notification)

	uc.log.Info("Successfully created notification", logFields(
		"notification_id", notification.ID.String(),
		"channel", notification.Channel.String(),
		"scheduled_at", notification.ScheduledAt,
	)...)

	return notification, nil
}

func validateRecipient(input input.CreateNotifyInput, channel vo.Channel) error {
	switch channel {
	case vo.Email:
		if input.EmailTo == nil || *input.EmailTo == "" {
			return fmt.Errorf("email required for email channel")
		}
		if input.TelegramID != nil || input.SmsTo != nil {
			return fmt.Errorf("only email should be provided for email channel")
		}
		if _, err := mail.ParseAddress(*input.EmailTo); err != nil {
			return fmt.Errorf("invalid email address: %w", err)
		}

	case vo.Telegram:
		if input.TelegramID == nil || *input.TelegramID == 0 {
			return fmt.Errorf("telegram chat ID required for telegram channel")
		}
		if input.EmailTo != nil || input.SmsTo != nil {
			return fmt.Errorf("only telegram chat ID should be provided for telegram channel")
		}

	case vo.SMS:
		if input.SmsTo == nil || *input.SmsTo == "" {
			return fmt.Errorf("phone number required for SMS channel")
		}
		if input.EmailTo != nil || input.TelegramID != nil {
			return fmt.Errorf("only phone number should be provided for SMS channel")
		}
	}
	return nil
}

func (uc *UseCase) Read(ctx context.Context, id string) (*model.Notification, error) {
	const op = "notification.UseCase.Read"
	logFields := logger.WithFields("operation", op, "notification_id", id)

	notificationID, err := parseID(op, id)
	if err != nil {
		return nil, err
	}

	notification, err := uc.cache.Get(ctx, notificationID)
	if err != nil {
		uc.log.Warn("Failed to read from cache", logFields("error", err.Error())...)
	}
	if notification != nil {
		uc.log.Debug("Notification read from cache", logFields()...)
		return notification, nil
	}

	notification, err = uc.repo.GetByID(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	uc.cw.Enqueue(notification)

	uc.log.Debug("Notification read from storage", logFields()...)
	return notification, nil
}

func (uc *UseCase) Cancel(ctx context.Context, id string) (*model.Notification, error) {
	const op = "notification.UseCase.Cancel"
	logFields := logger.WithFields("operation", op, "notification_id", id)

	notificationID, err := parseID(op, id)
	if err != nil {
		return nil, err
	}

	notification, err := uc.repo.Cancel(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	uc.cw.Enqueue(notification)

	uc.log.Info("Successfully canceled notification", logFields()...)
	return notification, nil
}

func parseID(op, id string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w: %w", op, errorx.ErrInvalidID, err)
	}
	return uid, nil
}

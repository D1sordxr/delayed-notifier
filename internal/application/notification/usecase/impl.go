package usecase

import (
	"context"
	"fmt"
	"time"
	"wb-tech-l3/internal/application/notification/input"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/params"
	"wb-tech-l3/internal/domain/core/notification/ports"
	"wb-tech-l3/internal/domain/core/notification/vo"
	"wb-tech-l3/pkg/logger"

	"github.com/google/uuid"
)

type UseCase struct {
	log  appPorts.Logger
	cs   ports.CacheStore
	repo ports.Repository
}

func NewUseCase(
	log appPorts.Logger,
	cs ports.CacheStore,
	repo ports.Repository,
) *UseCase {
	return &UseCase{
		log:  log,
		cs:   cs,
		repo: repo,
	}
}

func (uc *UseCase) Create(ctx context.Context, input input.CreateNotifyInput) (*model.Notification, error) {
	const op = "notification.UseCase.Create"
	logFields := logger.WithFields("operation", op)

	uc.log.Info("Attempting to create notification", logFields()...)

	channel, err := vo.ParseChannel(input.Channel)
	if err != nil {
		uc.log.Error("Error parsing channel", logFields("error", err.Error())...)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = uc.validateRecipient(input, channel); err != nil {
		uc.log.Error("Invalid recipient data", logFields("error", err.Error())...)
		return nil, fmt.Errorf("%s: %w", op, err)
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
		Attempts:       vo.DefaultAttempt,
		ScheduledAt:    input.Scheduled,
	})
	if err != nil {
		uc.log.Error("Error saving notification to database", logFields("error", err.Error())...)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err = uc.cs.Create(cacheCtx, notification); err != nil {
			uc.log.Warn("Error saving notification to cache", logFields("error", err.Error())...)
		}
	}()

	uc.log.Info("Successfully created notification", logFields(
		"notification_id", notification.ID.String(),
	)...)

	return notification, nil
}

func (uc *UseCase) validateRecipient(input input.CreateNotifyInput, channel vo.Channel) error {
	switch channel {
	case vo.Email:
		if input.EmailTo == nil || *input.EmailTo == "" {
			return fmt.Errorf("email required for email channel")
		}
		if input.TelegramID != nil || input.SmsTo != nil {
			return fmt.Errorf("only email should be provided for email channel")
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

	notificationID, err := uc.parseUUID(op, id)
	if err != nil {
		return nil, err
	}

	notification, err := uc.cs.Read(ctx, notificationID.String())
	if err != nil {
		uc.log.Warn("Failed to read from cache", logFields("error", err.Error())...)
	}
	if notification != nil {
		uc.log.Info("Successfully read notification from cache", logFields()...)
		return notification, nil
	}

	notification, err = uc.repo.GetByID(ctx, notificationID)
	if err != nil {
		uc.log.Error("Failed to read notification", logFields("error", err.Error())...)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err = uc.cs.Create(cacheCtx, notification); err != nil {
			uc.log.Warn("Error saving notification to cache", logFields("error", err.Error())...)
		}
	}()

	uc.log.Info("Successfully got notification from storage", logFields()...)
	return notification, nil
}

func (uc *UseCase) Delete(ctx context.Context, id string) error {
	const op = "notification.UseCase.Delete"
	logFields := logger.WithFields("operation", op, "notification_id", id)

	notificationID, err := uc.parseUUID(op, id)
	if err != nil {
		return err
	}

	if _, err = uc.repo.Cancel(ctx, notificationID); err != nil {
		uc.log.Error("Failed to cancel notification", logFields("error", err.Error())...)
		return fmt.Errorf("%s: %w", op, err)
	}

	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err = uc.cs.Delete(cacheCtx, notificationID.String()); err != nil {
			uc.log.Warn("Error deleting notification from cache", logFields("error", err.Error())...)
		}
	}()

	uc.log.Info("Successfully canceled notification", logFields()...)

	return nil
}

func (uc *UseCase) parseUUID(op, id string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		uc.log.Error("Error parsing UUID", "op", op, "id", id)
		return uid, fmt.Errorf("%s: error parsing UUID: %w", op, err)
	}
	return uid, nil
}

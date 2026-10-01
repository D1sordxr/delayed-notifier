package dispatcher

import (
	"context"
	"errors"
	"fmt"

	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/pkg/logger"

	"github.com/google/uuid"
)

// UseCase delivers a notification received from the broker.
//
// The notification row stays locked while it is being sent, so duplicates of
// a message (redelivery, re-claim of a stale one) are sent at most once per
// successful transaction: a concurrent duplicate waits for the lock and then
// finds the notification no longer pending.
type UseCase struct {
	log         appPorts.Logger
	tx          appPorts.TxManager
	repo        ports.Repository
	publisher   ports.Publisher
	sender      ports.Sender
	cw          ports.CacheWriter
	maxAttempts int16
}

func NewUseCase(
	log appPorts.Logger,
	tx appPorts.TxManager,
	repo ports.Repository,
	publisher ports.Publisher,
	sender ports.Sender,
	cw ports.CacheWriter,
	maxAttempts int16,
) *UseCase {
	if maxAttempts <= 0 {
		maxAttempts = vo.DefaultMaxAttempts
	}

	return &UseCase{
		log:         log,
		tx:          tx,
		repo:        repo,
		publisher:   publisher,
		sender:      sender,
		cw:          cw,
		maxAttempts: maxAttempts,
	}
}

func (uc *UseCase) Dispatch(ctx context.Context, id uuid.UUID) error {
	const op = "notification.Dispatcher.Dispatch"
	logFields := logger.WithFields("operation", op, "notification_id", id.String())

	var updated *model.Notification

	err := uc.tx.WithTransaction(ctx, func(ctx context.Context) error {
		n, err := uc.repo.LockPending(ctx, id)
		if errors.Is(err, errorx.ErrNotFound) {
			uc.log.Info("Notification is no longer pending, skipping", logFields()...)
			return nil
		}
		if err != nil {
			return err
		}

		sendErr := uc.sender.Send(ctx, n)
		switch {
		case sendErr == nil:
			updated, err = uc.repo.MarkSent(ctx, id)
			if err == nil {
				uc.log.Info("Notification sent", logFields("channel", n.Channel.String())...)
			}

		case n.Attempts+1 >= uc.maxAttempts:
			updated, err = uc.repo.MarkFailed(ctx, id)
			if err == nil {
				uc.log.Error("Notification failed, no attempts left",
					logFields("attempts", updated.Attempts, "error", sendErr.Error())...)
			}

		default:
			updated, err = uc.repo.MarkRetry(ctx, id)
			if err == nil {
				err = uc.publisher.PublishRetry(ctx, id)
			}
			if err == nil {
				uc.log.Warn("Notification delivery failed, retry scheduled",
					logFields("attempts", updated.Attempts, "error", sendErr.Error())...)
			}
		}

		return err
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if updated != nil {
		uc.cw.Enqueue(updated)
	}

	return nil
}

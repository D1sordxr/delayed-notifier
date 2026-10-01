package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/ports"

	"github.com/google/uuid"
)

type Config struct {
	// BatchSize is how many notifications one ScheduleDue call claims.
	BatchSize int32
	// Lookahead lets notifications due within this window be claimed early
	// and wait in the broker, so they are delivered on time instead of on
	// the next poll.
	Lookahead time.Duration
	// StaleAfter is how long a queued notification may stay pending before
	// it is claimed again (lost message, crashed dispatcher).
	StaleAfter time.Duration
}

// UseCase claims due notifications and hands them over to the broker.
// Claimed rows are locked with SKIP LOCKED, so any number of schedulers can
// run concurrently without publishing the same notification twice.
type UseCase struct {
	log       appPorts.Logger
	tx        appPorts.TxManager
	repo      ports.Repository
	publisher ports.Publisher
	cfg       Config
	now       func() time.Time
}

func NewUseCase(
	log appPorts.Logger,
	tx appPorts.TxManager,
	repo ports.Repository,
	publisher ports.Publisher,
	cfg Config,
) *UseCase {
	return &UseCase{
		log:       log,
		tx:        tx,
		repo:      repo,
		publisher: publisher,
		cfg:       cfg,
		now:       time.Now,
	}
}

func (uc *UseCase) ScheduleDue(ctx context.Context) (int, error) {
	const op = "notification.Scheduler.ScheduleDue"

	var (
		claimed    int
		publishErr error
	)

	err := uc.tx.WithTransaction(ctx, func(ctx context.Context) error {
		now := uc.now()

		due, err := uc.repo.ClaimDue(ctx, now.Add(uc.cfg.Lookahead), now.Add(-uc.cfg.StaleAfter), uc.cfg.BatchSize)
		if err != nil {
			return err
		}
		claimed = len(due)

		// Publishing happens while the rows are locked: a dispatcher that gets
		// a message before this transaction commits waits for the lock.
		queued := make([]uuid.UUID, 0, len(due))
		var errs []error
		for _, n := range due {
			if err = uc.publisher.Publish(ctx, n.ID, n.ScheduledAt.Sub(now)); err != nil {
				errs = append(errs, fmt.Errorf("notification %s: %w", n.ID, err))
				continue
			}
			queued = append(queued, n.ID)
		}
		// Not published notifications stay unqueued and are claimed again on the next run.
		publishErr = errors.Join(errs...)

		if len(queued) == 0 {
			return nil
		}
		return uc.repo.MarkQueued(ctx, queued)
	})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	if publishErr != nil {
		return claimed, fmt.Errorf("%s: publish: %w", op, publishErr)
	}

	if claimed > 0 {
		uc.log.Info("Scheduled due notifications", "operation", op, "count", claimed)
	}

	return claimed, nil
}

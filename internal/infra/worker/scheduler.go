package worker

import (
	"context"
	"sync"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/port"
	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
)

// Scheduler runs the scheduler use case on a ticker (cron.Handler).
// While batches come back full it runs again right away to catch up with
// a backlog instead of waiting for the next tick.
type Scheduler struct {
	log       appPorts.Logger
	uc        port.SchedulerUseCase
	interval  time.Duration
	batchSize int

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewScheduler(log appPorts.Logger, uc port.SchedulerUseCase, interval time.Duration, batchSize int) *Scheduler {
	return &Scheduler{
		log:       log,
		uc:        uc,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)

	s.wg.Go(func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			s.drain(ctx)

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	})

	return nil
}

func (s *Scheduler) drain(ctx context.Context) {
	for ctx.Err() == nil {
		claimed, err := s.uc.ScheduleDue(ctx)
		if err != nil {
			if ctx.Err() == nil {
				s.log.Error("Failed to schedule due notifications", "error", err.Error())
			}
			return
		}
		if claimed < s.batchSize {
			return
		}
	}
}

func (s *Scheduler) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

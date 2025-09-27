package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"

	"golang.org/x/sync/errgroup"
)

type Handler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Worker struct {
	log      ports.Logger
	handlers []Handler
}

func NewWorker(
	log ports.Logger,
	handlers ...Handler,
) *Worker {
	return &Worker{
		log:      log,
		handlers: handlers,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	w.log.Info("starting worker", "total_handlers", len(w.handlers))

	errChan := make(chan error, 1)
	errGroup, ctx := errgroup.WithContext(ctx)
	go func() { errChan <- errGroup.Wait() }()

	for _, handler := range w.handlers {
		func(handler Handler) {
			errGroup.Go(func() error {
				return handler.Start(ctx)
			})
		}(handler)
	}

	select {
	case err := <-errChan:
		w.log.Error("worker received critical error, initiating shutdown", "error", err.Error())
		return fmt.Errorf("worker error: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func (w *Worker) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	go func() {
		defer close(done)
		for i := len(w.handlers) - 1; i >= 0; i-- {
			if err := w.handlers[i].Stop(ctx); err != nil {
				w.log.Error("worker stopped with error", "error", err.Error())
			}
		}
	}()

	select {
	case <-done:
		w.log.Info("all handlers stopped")
		return nil
	case <-ctx.Done():
		w.log.Warn("forced shutdown due to context timeout")
		return ctx.Err()
	}
}

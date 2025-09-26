package handler

import (
	"context"
	"sync"
	"time"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/ports"

	"github.com/google/uuid"
)

const (
	batchSize       = 10
	chanBufferSize  = 256
	processInterval = 5 * time.Second
)

type NotificationWriter struct {
	log appPorts.Logger

	processor  ports.PendingProcessor
	publisher  ports.Publisher
	cacheStore ports.CacheStore

	cacheSetterQueue  chan *model.Notification
	cacheDeleterQueue chan uuid.UUID

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewNotificationWriter(
	log appPorts.Logger,
	processor ports.PendingProcessor,
	publisher ports.Publisher,
	cacheStore ports.CacheStore,
) *NotificationWriter {
	return &NotificationWriter{
		log:               log,
		processor:         processor,
		publisher:         publisher,
		cacheStore:        cacheStore,
		cacheSetterQueue:  make(chan *model.Notification, chanBufferSize),
		cacheDeleterQueue: make(chan uuid.UUID, chanBufferSize),
	}
}

func (w *NotificationWriter) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.startChannelWorkers(workerCtx)

	w.startProcessLoop(workerCtx)

	return nil
}

func (w *NotificationWriter) startChannelWorkers(ctx context.Context) {
	w.wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case notification, ok := <-w.cacheSetterQueue:
				if !ok {
					return
				}
				if err := w.cacheStore.Create(ctx, notification); err != nil {
					w.log.Warn("Failed to cache notification",
						"id", notification.ID.String(), "error", err,
					)
				}
			}
		}
	})

	w.wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case id, ok := <-w.cacheDeleterQueue:
				if !ok {
					return
				}
				if err := w.cacheStore.Delete(ctx, id.String()); err != nil {
					w.log.Warn("Failed to delete cache",
						"id", id.String(), "error", err)
				}
			}
		}
	})
}

func (w *NotificationWriter) startProcessLoop(ctx context.Context) {
	w.wg.Go(func() {
		ticker := time.NewTicker(processInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.processBatch(ctx)
			}
		}
	})
}

func (w *NotificationWriter) processBatch(ctx context.Context) {
	err := w.processor.ProcessPending(
		ctx,
		batchSize,
		func(ctx context.Context, notification *model.Notification) error {
			if err := w.publisher.Publish(ctx, notification); err != nil {
				w.log.Error("Failed to publish notification",
					"id", notification.ID, "error", err.Error(),
				)

				select {
				case w.cacheDeleterQueue <- notification.ID:
				case <-ctx.Done():
					return ctx.Err()
				default:
					w.log.Warn("Cache deleter queue full, dropping delete request")
				}
				return err
			}

			select {
			case w.cacheSetterQueue <- notification:
			case <-ctx.Done():
				return ctx.Err()
			default:
				w.log.Warn("Cache setter queue full, dropping cache update")
			}
			return nil
		})

	if err != nil {
		w.log.Error("Error processing pending notifications", "error", err.Error())
	}
}

func (w *NotificationWriter) Stop(ctx context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}

	close(w.cacheSetterQueue)
	close(w.cacheDeleterQueue)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.log.Info("Notification worker stopped gracefully")
		return nil
	case <-ctx.Done():
		w.log.Warn("Notification worker stop timed out")
		return ctx.Err()
	}
}

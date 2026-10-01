package notification

import (
	"context"
	"sync"
	"time"

	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/ports"
)

const writeTimeout = time.Second

// Writer updates the cache off the request path (app.Component).
// Enqueue never blocks: when the buffer is full the write is dropped and the
// next read falls through to the database.
type Writer struct {
	log     appPorts.Logger
	store   ports.CacheStore
	queue   chan *model.Notification
	workers int

	finished chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
}

func NewWriter(log appPorts.Logger, store ports.CacheStore, bufferSize, workers int) *Writer {
	return &Writer{
		log:      log,
		store:    store,
		queue:    make(chan *model.Notification, max(bufferSize, 1)),
		workers:  max(workers, 1),
		finished: make(chan struct{}),
		stop:     make(chan struct{}),
	}
}

func (w *Writer) Enqueue(n *model.Notification) {
	select {
	case w.queue <- n:
	default:
		w.log.Warn("Cache write buffer is full, dropping write", "notification_id", n.ID.String())
	}
}

// Start writes queued notifications until ctx is cancelled or Shutdown is
// called, then flushes what is left in the buffer.
func (w *Writer) Start(ctx context.Context) error {
	defer close(w.finished)

	var wg sync.WaitGroup
	for range w.workers {
		wg.Go(func() {
			for {
				select {
				case n := <-w.queue:
					w.write(n)
				case <-ctx.Done():
					w.flush()
					return
				case <-w.stop:
					w.flush()
					return
				}
			}
		})
	}
	wg.Wait()

	return nil
}

func (w *Writer) flush() {
	for {
		select {
		case n := <-w.queue:
			w.write(n)
		default:
			return
		}
	}
}

func (w *Writer) write(n *model.Notification) {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	if err := w.store.Set(ctx, n); err != nil {
		w.log.Warn("Failed to write notification to cache",
			"notification_id", n.ID.String(), "error", err.Error())
	}
}

func (w *Writer) Shutdown(ctx context.Context) error {
	w.stopOnce.Do(func() { close(w.stop) })

	select {
	case <-w.finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

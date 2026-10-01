// Package fakes provides in-memory port implementations for use case tests.
package fakes

import (
	"context"
	"sync"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"

	"github.com/google/uuid"
)

type Logger struct{}

func (Logger) Info(string, ...any)  {}
func (Logger) Error(string, ...any) {}
func (Logger) Warn(string, ...any)  {}
func (Logger) Debug(string, ...any) {}

// Tx runs fn directly; a returned error is passed through.
type Tx struct{}

func (Tx) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// Repository keeps notifications in memory. Claim and lock semantics are
// simplified: there is no concurrency, only the status/queued filters.
type Repository struct {
	mu    sync.Mutex
	Items map[uuid.UUID]*model.Notification
	// Queued records MarkQueued calls.
	Queued []uuid.UUID
	Err    error
}

func NewRepository(items ...*model.Notification) *Repository {
	r := &Repository{Items: map[uuid.UUID]*model.Notification{}}
	for _, n := range items {
		r.Items[n.ID] = n
	}
	return r
}

func (r *Repository) Create(_ context.Context, p params.CreateNotificationParams) (*model.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Err != nil {
		return nil, r.Err
	}

	now := time.Now()
	n := &model.Notification{
		ID: uuid.New(), Subject: p.Subject, Message: p.Message, AuthorID: p.AuthorID,
		EmailTo: p.EmailTo, TelegramChatID: p.TelegramChatID, SmsTo: p.SmsTo,
		Channel: p.Channel, Status: p.Status, Attempts: p.Attempts,
		ScheduledAt: p.ScheduledAt, CreatedAt: now, UpdatedAt: now,
	}
	r.Items[n.ID] = n
	return clone(n), nil
}

func (r *Repository) GetByID(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.Items[id]
	if !ok {
		return nil, errorx.ErrNotFound
	}
	return clone(n), nil
}

func (r *Repository) Cancel(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.Items[id]
	switch {
	case !ok:
		return nil, errorx.ErrNotFound
	case n.Status == vo.Pending:
		n.Status = vo.Declined
	case n.Status != vo.Declined:
		return nil, errorx.ErrNotCancellable
	}
	return clone(n), nil
}

func (r *Repository) ClaimDue(_ context.Context, dueBefore, _ time.Time, limit int32) ([]*model.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Err != nil {
		return nil, r.Err
	}

	var due []*model.Notification
	for _, n := range r.Items {
		if n.Status == vo.Pending && !n.ScheduledAt.After(dueBefore) && len(due) < int(limit) {
			due = append(due, clone(n))
		}
	}
	return due, nil
}

func (r *Repository) MarkQueued(_ context.Context, ids []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Queued = append(r.Queued, ids...)
	return nil
}

func (r *Repository) LockPending(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.Items[id]
	if !ok || n.Status != vo.Pending {
		return nil, errorx.ErrNotFound
	}
	return clone(n), nil
}

func (r *Repository) MarkSent(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.update(id, func(n *model.Notification) {
		now := time.Now()
		n.Status, n.SentAt = vo.Sent, &now
	})
}

func (r *Repository) MarkFailed(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.update(id, func(n *model.Notification) { n.Status = vo.Failed })
}

func (r *Repository) MarkRetry(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.update(id, func(*model.Notification) {})
}

func (r *Repository) update(id uuid.UUID, fn func(*model.Notification)) (*model.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.Items[id]
	if !ok {
		return nil, errorx.ErrNotFound
	}
	fn(n)
	n.Attempts++
	n.UpdatedAt = time.Now()
	return clone(n), nil
}

func clone(n *model.Notification) *model.Notification {
	c := *n
	return &c
}

type Published struct {
	ID    uuid.UUID
	Delay time.Duration
	Retry bool
}

// Publisher records publications; FailFor makes Publish fail for given ids.
type Publisher struct {
	mu        sync.Mutex
	Published []Published
	FailFor   map[uuid.UUID]error
}

func (p *Publisher) Publish(_ context.Context, id uuid.UUID, delay time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.FailFor[id]; err != nil {
		return err
	}
	p.Published = append(p.Published, Published{ID: id, Delay: delay})
	return nil
}

func (p *Publisher) PublishRetry(_ context.Context, id uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.FailFor[id]; err != nil {
		return err
	}
	p.Published = append(p.Published, Published{ID: id, Retry: true})
	return nil
}

// Sender returns Err for every call and counts calls.
type Sender struct {
	Err   error
	Calls int
}

func (s *Sender) Send(context.Context, *model.Notification) error {
	s.Calls++
	return s.Err
}

// Cache is an in-memory CacheStore and CacheWriter: Enqueue writes through.
type Cache struct {
	mu       sync.Mutex
	Items    map[uuid.UUID]*model.Notification
	Enqueued []*model.Notification
	GetErr   error
}

func NewCache() *Cache {
	return &Cache{Items: map[uuid.UUID]*model.Notification{}}
}

func (c *Cache) Get(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.GetErr != nil {
		return nil, c.GetErr
	}
	return c.Items[id], nil
}

func (c *Cache) Set(_ context.Context, n *model.Notification) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Items[n.ID] = n
	return nil
}

func (c *Cache) Enqueue(n *model.Notification) {
	c.mu.Lock()
	c.Enqueued = append(c.Enqueued, n)
	c.mu.Unlock()

	_ = c.Set(context.Background(), n)
}

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/fakes"
	"github.com/D1sordxr/delayed-notifier/internal/application/notification/input"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func newUseCase(items ...*model.Notification) (*UseCase, *fakes.Repository, *fakes.Cache) {
	repo := fakes.NewRepository(items...)
	cache := fakes.NewCache()
	return NewUseCase(fakes.Logger{}, cache, cache, repo), repo, cache
}

func TestCreate(t *testing.T) {
	t.Parallel()

	scheduled := time.Now().Add(time.Hour)
	tests := []struct {
		name    string
		in      input.CreateNotifyInput
		wantErr error
	}{
		{
			name: "email",
			in:   input.CreateNotifyInput{Channel: "email", EmailTo: ptr("user@example.com")},
		},
		{
			name: "telegram",
			in:   input.CreateNotifyInput{Channel: "telegram", TelegramID: ptr[int64](42)},
		},
		{
			name: "sms",
			in:   input.CreateNotifyInput{Channel: "sms", SmsTo: ptr("+10000000000")},
		},
		{
			name:    "unknown channel",
			in:      input.CreateNotifyInput{Channel: "pigeon"},
			wantErr: errorx.ErrInvalidChannel,
		},
		{
			name:    "missing recipient",
			in:      input.CreateNotifyInput{Channel: "email"},
			wantErr: errorx.ErrInvalidRecipient,
		},
		{
			name:    "invalid email",
			in:      input.CreateNotifyInput{Channel: "email", EmailTo: ptr("not-an-email")},
			wantErr: errorx.ErrInvalidRecipient,
		},
		{
			name:    "recipient of another channel",
			in:      input.CreateNotifyInput{Channel: "sms", SmsTo: ptr("+1"), EmailTo: ptr("user@example.com")},
			wantErr: errorx.ErrInvalidRecipient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc, repo, cache := newUseCase()
			tt.in.AuthorID, tt.in.Subject, tt.in.Message, tt.in.Scheduled = "author", "subject", "message", scheduled

			n, err := uc.Create(context.Background(), tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if len(repo.Items) != 0 {
					t.Fatal("invalid notification was stored")
				}
				return
			}

			if n.Status != vo.Pending || n.Attempts != 0 || !n.ScheduledAt.Equal(scheduled) {
				t.Fatalf("created = %+v, want pending with 0 attempts", n)
			}
			if len(cache.Enqueued) != 1 {
				t.Fatalf("cache writes = %d, want 1", len(cache.Enqueued))
			}
		})
	}
}

func TestReadUsesCacheThenStorage(t *testing.T) {
	t.Parallel()

	stored := &model.Notification{ID: uuid.New(), Subject: "stored", Status: vo.Pending}
	uc, _, cache := newUseCase(stored)

	got, err := uc.Read(context.Background(), stored.ID.String())
	if err != nil || got.Subject != "stored" {
		t.Fatalf("Read() = %v, %v; want stored notification", got, err)
	}
	if len(cache.Enqueued) != 1 {
		t.Fatalf("cache miss did not populate the cache")
	}

	cache.Items[stored.ID] = &model.Notification{ID: stored.ID, Subject: "cached"}
	got, err = uc.Read(context.Background(), stored.ID.String())
	if err != nil || got.Subject != "cached" {
		t.Fatalf("Read() = %v, %v; want cached notification", got, err)
	}
}

func TestReadFallsBackToStorageWhenCacheFails(t *testing.T) {
	t.Parallel()

	stored := &model.Notification{ID: uuid.New(), Subject: "stored"}
	uc, _, cache := newUseCase(stored)
	cache.GetErr = errors.New("redis down")

	got, err := uc.Read(context.Background(), stored.ID.String())
	if err != nil || got.Subject != "stored" {
		t.Fatalf("Read() = %v, %v; want stored notification", got, err)
	}
}

func TestReadErrors(t *testing.T) {
	t.Parallel()

	uc, _, _ := newUseCase()

	if _, err := uc.Read(context.Background(), "not-a-uuid"); !errors.Is(err, errorx.ErrInvalidID) {
		t.Errorf("Read(invalid) error = %v, want %v", err, errorx.ErrInvalidID)
	}
	if _, err := uc.Read(context.Background(), uuid.NewString()); !errors.Is(err, errorx.ErrNotFound) {
		t.Errorf("Read(missing) error = %v, want %v", err, errorx.ErrNotFound)
	}
}

func TestCancel(t *testing.T) {
	t.Parallel()

	pending := &model.Notification{ID: uuid.New(), Status: vo.Pending}
	sent := &model.Notification{ID: uuid.New(), Status: vo.Sent}
	uc, _, cache := newUseCase(pending, sent)

	got, err := uc.Cancel(context.Background(), pending.ID.String())
	if err != nil || got.Status != vo.Declined {
		t.Fatalf("Cancel(pending) = %v, %v; want declined", got, err)
	}
	if cached := cache.Items[pending.ID]; cached == nil || cached.Status != vo.Declined {
		t.Fatalf("cache was not updated with the declined state")
	}

	if _, err = uc.Cancel(context.Background(), pending.ID.String()); err != nil {
		t.Errorf("Cancel(declined) error = %v, want nil (idempotent)", err)
	}
	if _, err = uc.Cancel(context.Background(), sent.ID.String()); !errors.Is(err, errorx.ErrNotCancellable) {
		t.Errorf("Cancel(sent) error = %v, want %v", err, errorx.ErrNotCancellable)
	}
	if _, err = uc.Cancel(context.Background(), uuid.NewString()); !errors.Is(err, errorx.ErrNotFound) {
		t.Errorf("Cancel(missing) error = %v, want %v", err, errorx.ErrNotFound)
	}
	if _, err = uc.Cancel(context.Background(), "bad"); !errors.Is(err, errorx.ErrInvalidID) {
		t.Errorf("Cancel(invalid) error = %v, want %v", err, errorx.ErrInvalidID)
	}
}

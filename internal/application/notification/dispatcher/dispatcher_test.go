package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/fakes"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"

	"github.com/google/uuid"
)

func TestDispatch(t *testing.T) {
	t.Parallel()

	sendErr := errors.New("smtp unavailable")

	tests := []struct {
		name        string
		status      vo.Status
		attempts    int16
		sendErr     error
		wantStatus  vo.Status
		wantSends   int
		wantRetry   bool
		wantCached  bool
		wantAttempt int16
	}{
		{name: "sent", status: vo.Pending, wantStatus: vo.Sent, wantSends: 1, wantCached: true, wantAttempt: 1},
		{name: "retry", status: vo.Pending, attempts: 0, sendErr: sendErr, wantStatus: vo.Pending, wantSends: 1, wantRetry: true, wantCached: true, wantAttempt: 1},
		{name: "last attempt fails", status: vo.Pending, attempts: 2, sendErr: sendErr, wantStatus: vo.Failed, wantSends: 1, wantCached: true, wantAttempt: 3},
		{name: "cancelled is skipped", status: vo.Declined, wantStatus: vo.Declined},
		{name: "already sent is skipped", status: vo.Sent, attempts: 1, wantStatus: vo.Sent, wantAttempt: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := &model.Notification{ID: uuid.New(), Channel: vo.Email, Status: tt.status, Attempts: tt.attempts}
			repo := fakes.NewRepository(n)
			pub := &fakes.Publisher{}
			sender := &fakes.Sender{Err: tt.sendErr}
			cache := fakes.NewCache()

			uc := NewUseCase(fakes.Logger{}, fakes.Tx{}, repo, pub, sender, cache, 3)
			if err := uc.Dispatch(context.Background(), n.ID); err != nil {
				t.Fatalf("Dispatch() = %v", err)
			}

			got := repo.Items[n.ID]
			if got.Status != tt.wantStatus || got.Attempts != tt.wantAttempt {
				t.Errorf("state = %s/%d attempts, want %s/%d", got.Status, got.Attempts, tt.wantStatus, tt.wantAttempt)
			}
			if sender.Calls != tt.wantSends {
				t.Errorf("sends = %d, want %d", sender.Calls, tt.wantSends)
			}
			if retried := len(pub.Published) == 1 && pub.Published[0].Retry; retried != tt.wantRetry {
				t.Errorf("retry published = %v, want %v", retried, tt.wantRetry)
			}
			if cached := len(cache.Enqueued) > 0; cached != tt.wantCached {
				t.Errorf("cache updated = %v, want %v", cached, tt.wantCached)
			}
		})
	}
}

func TestDispatchFailsWhenRetryCannotBePublished(t *testing.T) {
	t.Parallel()

	n := &model.Notification{ID: uuid.New(), Channel: vo.SMS, Status: vo.Pending}
	boom := errors.New("broker down")
	uc := NewUseCase(
		fakes.Logger{}, fakes.Tx{}, fakes.NewRepository(n),
		&fakes.Publisher{FailFor: map[uuid.UUID]error{n.ID: boom}},
		&fakes.Sender{Err: errors.New("gateway error")},
		fakes.NewCache(), 3,
	)

	// The error rejects the delivery; with a real transaction the attempt
	// is rolled back and the notification is re-claimed once stale.
	if err := uc.Dispatch(context.Background(), n.ID); !errors.Is(err, boom) {
		t.Fatalf("Dispatch() error = %v, want %v", err, boom)
	}
}

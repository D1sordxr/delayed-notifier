package scheduler

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/fakes"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"

	"github.com/google/uuid"
)

func pending(scheduledAt time.Time) *model.Notification {
	return &model.Notification{ID: uuid.New(), Status: vo.Pending, ScheduledAt: scheduledAt}
}

func TestScheduleDue(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	overdue := pending(now.Add(-time.Minute))
	soon := pending(now.Add(500 * time.Millisecond))
	later := pending(now.Add(time.Hour))
	cancelled := pending(now.Add(-time.Minute))
	cancelled.Status = vo.Declined

	repo := fakes.NewRepository(overdue, soon, later, cancelled)
	pub := &fakes.Publisher{}
	uc := NewUseCase(fakes.Logger{}, fakes.Tx{}, repo, pub, Config{BatchSize: 10, Lookahead: time.Second, StaleAfter: time.Minute})
	uc.now = func() time.Time { return now }

	claimed, err := uc.ScheduleDue(context.Background())
	if err != nil {
		t.Fatalf("ScheduleDue() = %v", err)
	}
	if claimed != 2 {
		t.Fatalf("claimed = %d, want 2", claimed)
	}

	delays := map[uuid.UUID]time.Duration{}
	for _, p := range pub.Published {
		delays[p.ID] = p.Delay
	}
	if d, ok := delays[overdue.ID]; !ok || d > 0 {
		t.Errorf("overdue delay = %v (published %v), want immediate", d, ok)
	}
	if d := delays[soon.ID]; d != 500*time.Millisecond {
		t.Errorf("soon delay = %v, want 500ms", d)
	}
	if _, ok := delays[later.ID]; ok {
		t.Error("notification outside the look-ahead window was published")
	}

	slices.SortFunc(repo.Queued, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	want := []uuid.UUID{overdue.ID, soon.ID}
	slices.SortFunc(want, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	if !slices.Equal(repo.Queued, want) {
		t.Errorf("queued = %v, want %v", repo.Queued, want)
	}
}

func TestScheduleDueMarksOnlyPublished(t *testing.T) {
	t.Parallel()

	ok, broken := pending(time.Now().Add(-time.Second)), pending(time.Now().Add(-time.Second))
	repo := fakes.NewRepository(ok, broken)
	boom := errors.New("broker down")
	pub := &fakes.Publisher{FailFor: map[uuid.UUID]error{broken.ID: boom}}
	uc := NewUseCase(fakes.Logger{}, fakes.Tx{}, repo, pub, Config{BatchSize: 10, StaleAfter: time.Minute})

	claimed, err := uc.ScheduleDue(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("ScheduleDue() error = %v, want %v", err, boom)
	}
	if claimed != 2 {
		t.Errorf("claimed = %d, want 2", claimed)
	}
	if !slices.Equal(repo.Queued, []uuid.UUID{ok.ID}) {
		t.Errorf("queued = %v, want only %v", repo.Queued, ok.ID)
	}
}

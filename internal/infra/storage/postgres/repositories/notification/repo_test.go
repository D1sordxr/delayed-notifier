package notification_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification"

	pgPool "github.com/D1sordxr/packages/postgres"
	exec "github.com/D1sordxr/packages/postgres/executor"
	"github.com/D1sordxr/packages/postgres/tx"
	"github.com/google/uuid"
)

// Integration tests: they run only when a test database DSN is set.
// The database is migrated and the notifications table is truncated.
const dsnEnv = "DELAYED_NOTIFIER_TEST_POSTGRES_DSN"

type fixture struct {
	repo *notification.Repository
	tx   *tx.ManagerImpl
}

func setup(t *testing.T) fixture {
	t.Helper()

	dsn := os.Getenv(dsnEnv)
	if dsn == "" {
		t.Skipf("%s is not set", dsnEnv)
	}

	ctx := context.Background()
	pool, err := pgPool.NewPool(ctx, &pgPool.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("NewPool() = %v", err)
	}
	t.Cleanup(pool.Close)

	if err = postgres.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() = %v", err)
	}
	if _, err = pool.Exec(ctx, `TRUNCATE notifications`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	executor := exec.NewExecutor(pool)
	return fixture{repo: notification.NewRepository(executor), tx: tx.NewManager(executor)}
}

func (f fixture) create(t *testing.T, scheduledAt time.Time) uuid.UUID {
	t.Helper()

	email := "user@example.com"
	n, err := f.repo.Create(context.Background(), params.CreateNotificationParams{
		Subject: "s", Message: "m", EmailTo: &email,
		Channel: vo.Email, Status: vo.Pending, ScheduledAt: scheduledAt,
	})
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	return n.ID
}

func (f fixture) claim(t *testing.T, ctx context.Context, limit int32) []uuid.UUID {
	t.Helper()

	now := time.Now()
	due, err := f.repo.ClaimDue(ctx, now, now.Add(-time.Minute), limit)
	if err != nil {
		t.Fatalf("ClaimDue() = %v", err)
	}

	ids := make([]uuid.UUID, len(due))
	for i, n := range due {
		ids[i] = n.ID
	}
	return ids
}

func TestCreateAndGet(t *testing.T) {
	f := setup(t)
	id := f.create(t, time.Now())

	n, err := f.repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByID() = %v", err)
	}
	if n.Status != vo.Pending || n.Channel != vo.Email || n.SmsTo != nil || n.SentAt != nil || *n.EmailTo != "user@example.com" {
		t.Fatalf("got %+v", n)
	}

	if _, err = f.repo.GetByID(context.Background(), uuid.New()); !errors.Is(err, errorx.ErrNotFound) {
		t.Fatalf("GetByID(missing) = %v, want %v", err, errorx.ErrNotFound)
	}
}

func TestClaimDueSkipsLockedAndQueued(t *testing.T) {
	f := setup(t)
	past := time.Now().Add(-time.Minute)
	first, second := f.create(t, past), f.create(t, past.Add(time.Second))
	f.create(t, time.Now().Add(time.Hour)) // not due

	err := f.tx.WithTransaction(context.Background(), func(ctx context.Context) error {
		claimed := f.claim(t, ctx, 1)
		if !slices.Equal(claimed, []uuid.UUID{first}) {
			t.Errorf("first claim = %v, want oldest %v", claimed, first)
		}

		// A concurrent scheduler skips the row locked by this transaction.
		other := f.claim(t, context.Background(), 10)
		if !slices.Equal(other, []uuid.UUID{second}) {
			t.Errorf("concurrent claim = %v, want %v", other, second)
		}

		return f.repo.MarkQueued(ctx, claimed)
	})
	if err != nil {
		t.Fatalf("WithTransaction() = %v", err)
	}

	// The queued notification is not claimed again until it is stale.
	if again := f.claim(t, context.Background(), 10); !slices.Equal(again, []uuid.UUID{second}) {
		t.Errorf("claim after queue = %v, want %v", again, second)
	}
	now := time.Now()
	stale, err := f.repo.ClaimDue(context.Background(), now, now.Add(time.Minute), 10)
	if err != nil || len(stale) != 2 {
		t.Errorf("claim with stale cutoff in the future = %d rows, %v; want 2", len(stale), err)
	}
}

func TestDeliveryLifecycle(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.create(t, time.Now())

	created, _ := f.repo.GetByID(ctx, id)

	retried, err := f.repo.MarkRetry(ctx, id)
	if err != nil || retried.Attempts != 1 || retried.Status != vo.Pending {
		t.Fatalf("MarkRetry() = %+v, %v", retried, err)
	}
	if !retried.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updated_at did not advance: %v -> %v", created.UpdatedAt, retried.UpdatedAt)
	}

	locked, err := f.repo.LockPending(ctx, id)
	if err != nil || locked.ID != id {
		t.Fatalf("LockPending() = %v, %v", locked, err)
	}

	sent, err := f.repo.MarkSent(ctx, id)
	if err != nil || sent.Status != vo.Sent || sent.Attempts != 2 || sent.SentAt == nil {
		t.Fatalf("MarkSent() = %+v, %v", sent, err)
	}

	if _, err = f.repo.LockPending(ctx, id); !errors.Is(err, errorx.ErrNotFound) {
		t.Errorf("LockPending(sent) = %v, want %v", err, errorx.ErrNotFound)
	}
	if _, err = f.repo.Cancel(ctx, id); !errors.Is(err, errorx.ErrNotCancellable) {
		t.Errorf("Cancel(sent) = %v, want %v", err, errorx.ErrNotCancellable)
	}
}

func TestCancel(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.create(t, time.Now().Add(time.Hour))

	cancelled, err := f.repo.Cancel(ctx, id)
	if err != nil || cancelled.Status != vo.Declined {
		t.Fatalf("Cancel() = %+v, %v", cancelled, err)
	}
	if again, err := f.repo.Cancel(ctx, id); err != nil || again.Status != vo.Declined {
		t.Fatalf("second Cancel() = %+v, %v; want idempotent", again, err)
	}
	if _, err = f.repo.Cancel(ctx, uuid.New()); !errors.Is(err, errorx.ErrNotFound) {
		t.Fatalf("Cancel(missing) = %v, want %v", err, errorx.ErrNotFound)
	}
	if _, err = f.repo.LockPending(ctx, id); !errors.Is(err, errorx.ErrNotFound) {
		t.Fatalf("LockPending(declined) = %v, want %v", err, errorx.ErrNotFound)
	}
}

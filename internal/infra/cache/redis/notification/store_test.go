package notification_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/internal/infra/cache/redis/notification"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Integration tests: they run only when a Redis address is set.
const addrEnv = "DELAYED_NOTIFIER_TEST_REDIS_ADDR"

func newStore(t *testing.T) *notification.Store {
	t.Helper()

	addr := os.Getenv(addrEnv)
	if addr == "" {
		t.Skipf("%s is not set", addrEnv)
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	return notification.NewStore(client, time.Minute)
}

func TestStoreMiss(t *testing.T) {
	store := newStore(t)

	n, err := store.Get(context.Background(), uuid.New())
	if err != nil || n != nil {
		t.Fatalf("Get(missing) = %v, %v; want nil, nil", n, err)
	}
}

func TestStoreKeepsNewestState(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	base := time.Now().Truncate(time.Microsecond)
	pending := &model.Notification{ID: uuid.New(), Status: vo.Pending, UpdatedAt: base}
	sent := *pending
	sent.Status, sent.UpdatedAt = vo.Sent, base.Add(time.Millisecond)

	if err := store.Set(ctx, pending); err != nil {
		t.Fatalf("Set(pending) = %v", err)
	}
	if err := store.Set(ctx, &sent); err != nil {
		t.Fatalf("Set(sent) = %v", err)
	}
	// A stale write arriving late must not overwrite the newer state.
	if err := store.Set(ctx, pending); err != nil {
		t.Fatalf("Set(stale) = %v", err)
	}

	got, err := store.Get(ctx, pending.ID)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if got.Status != vo.Sent || !got.UpdatedAt.Equal(sent.UpdatedAt) {
		t.Fatalf("cached = %s at %v, want sent at %v", got.Status, got.UpdatedAt, sent.UpdatedAt)
	}
}

package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix    = "notification:"
	fieldData    = "data"
	fieldVersion = "version"
)

// setIfNewer stores the notification only if the cached version is not newer.
// Writes arrive asynchronously from several services, so a stale state read
// before an update could otherwise overwrite the fresh one.
// Versions are updated_at in microseconds: they fit a Lua number exactly.
var setIfNewer = redis.NewScript(`
local current = redis.call('HGET', KEYS[1], 'version')
if current and tonumber(current) > tonumber(ARGV[1]) then
	return 0
end
redis.call('HSET', KEYS[1], 'version', ARGV[1], 'data', ARGV[2])
redis.call('PEXPIRE', KEYS[1], ARGV[3])
return 1
`)

type Store struct {
	client redis.Cmdable
	ttl    time.Duration
}

func NewStore(client redis.Cmdable, ttl time.Duration) *Store {
	return &Store{client: client, ttl: ttl}
}

func key(id uuid.UUID) string {
	return keyPrefix + id.String()
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	const op = "redis.notification.Store.Get"

	data, err := s.client.HGet(ctx, key(id), fieldData).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var notification model.Notification
	if err = json.Unmarshal(data, &notification); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &notification, nil
}

func (s *Store) Set(ctx context.Context, n *model.Notification) error {
	const op = "redis.notification.Store.Set"

	data, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = setIfNewer.Run(ctx, s.client,
		[]string{key(n.ID)},
		n.UpdatedAt.UnixMicro(), data, s.ttl.Milliseconds(),
	).Err()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

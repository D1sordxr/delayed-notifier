package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"

	"github.com/wb-go/wbf/redis"
)

type Adapter struct {
	client *redis.Client
}

func NewAdapter(client *redis.Client) *Adapter {
	return &Adapter{client: client}
}

func (s *Adapter) Create(ctx context.Context, notification *model.Notification) error {
	const op = "redis.notification.Adapter.Create"

	data, err := json.Marshal(&notification)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if err = s.client.Set(
		ctx,
		vo.WithStorageKeyPrefix(notification.ID.String()),
		data,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Adapter) Read(ctx context.Context, id string) (*model.Notification, error) {
	const op = "redis.notification.Adapter.Read"

	result, err := s.client.Get(ctx, vo.WithStorageKeyPrefix(id))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var notification model.Notification
	if err = json.Unmarshal([]byte(result), &notification); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &notification, nil
}

func (s *Adapter) Delete(ctx context.Context, id string) error {
	const op = "redis.notification.Adapter.Delete"

	if err := s.client.Del(ctx, vo.WithStorageKeyPrefix(id)).Err(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Adapter) SetDeleted(ctx context.Context, id string) error {
	const op = "redis.notification.Adapter.SetDeleted"

	if err := s.client.Set(ctx, vo.WithStorageKeyPrefixDeleted(id), vo.DeletedValue); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_ = s.client.Del(ctx, vo.WithStorageKeyPrefix(id))
	return nil
}

func (s *Adapter) IsDeleted(ctx context.Context, id string) (bool, error) {
	const op = "redis.notification.Adapter.IsDeleted"

	result, err := s.client.Get(ctx, vo.WithStorageKeyPrefixDeleted(id))
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if result == vo.DeletedValue {
		return true, nil
	}
	return false, nil
}

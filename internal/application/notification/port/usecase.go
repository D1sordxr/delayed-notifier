package port

import (
	"context"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/input"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"

	"github.com/google/uuid"
)

type NotifyUseCase interface {
	Create(ctx context.Context, notify input.CreateNotifyInput) (*model.Notification, error)
	Read(ctx context.Context, id string) (*model.Notification, error)
	Cancel(ctx context.Context, id string) (*model.Notification, error)
}

type SchedulerUseCase interface {
	// ScheduleDue hands one batch of due notifications over to the broker
	// and returns how many were claimed.
	ScheduleDue(ctx context.Context) (int, error)
}

type DispatcherUseCase interface {
	Dispatch(ctx context.Context, id uuid.UUID) error
}

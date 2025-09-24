package ports

import (
	"context"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/params"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, p params.CreateNotificationParams) (*model.Notification, error)
	Cancel(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	UpdateStatus(ctx context.Context, p params.UpdateNotificationStatusParams) (*model.Notification, error)
}

type PendingProcessor interface {
	ProcessPending(
		ctx context.Context,
		batchSize int32,
		processor func(ctx context.Context, m *model.Notification) error,
	) error
}

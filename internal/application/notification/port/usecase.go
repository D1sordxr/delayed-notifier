package port

import (
	"context"
	"wb-tech-l3/internal/application/notification/input"
	"wb-tech-l3/internal/domain/core/notification/model"
)

type NotifyUseCase interface {
	Create(ctx context.Context, notify input.CreateNotifyInput) (*model.Notification, error)
	Read(ctx context.Context, id string) (*model.Notification, error)
	Delete(ctx context.Context, id string) error
}

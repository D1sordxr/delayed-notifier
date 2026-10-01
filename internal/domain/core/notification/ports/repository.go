package ports

import (
	"context"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"

	"github.com/google/uuid"
)

// Repository is the notification storage. Methods that change a single
// notification return its new state. Missing rows are reported as
// errorx.ErrNotFound.
type Repository interface {
	Create(ctx context.Context, p params.CreateNotificationParams) (*model.Notification, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	// Cancel declines a pending notification. A notification that is already
	// declined is returned as is; one that is sent or failed gives
	// errorx.ErrNotCancellable.
	Cancel(ctx context.Context, id uuid.UUID) (*model.Notification, error)

	// ClaimDue locks, within the current transaction, up to limit pending
	// notifications scheduled before dueBefore that were not queued, or were
	// queued before staleBefore. Rows locked by other transactions are skipped.
	ClaimDue(ctx context.Context, dueBefore, staleBefore time.Time, limit int32) ([]*model.Notification, error)
	MarkQueued(ctx context.Context, ids []uuid.UUID) error

	// LockPending locks a pending notification within the current transaction,
	// waiting for a concurrent sender to finish. errorx.ErrNotFound means it
	// is no longer pending (sent, failed, cancelled) or does not exist.
	LockPending(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	MarkSent(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	MarkFailed(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	MarkRetry(ctx context.Context, id uuid.UUID) (*model.Notification, error)
}

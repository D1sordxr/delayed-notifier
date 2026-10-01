package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/params"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification/converters"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification/gen"

	exec "github.com/D1sordxr/packages/postgres/executor"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository runs queries on the executor bound to ctx, so its methods take
// part in a transaction started by the transaction manager.
type Repository struct {
	q exec.Querier
}

func NewRepository(q exec.Querier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) queries(ctx context.Context) *gen.Queries {
	return gen.New(r.q.GetExecutor(ctx))
}

func (r *Repository) Create(ctx context.Context, p params.CreateNotificationParams) (*model.Notification, error) {
	const op = "postgres.notification.Repository.Create"

	rawModel, err := r.queries(ctx).CreateNotification(ctx, converters.ConvertCreateParams(p))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	const op = "postgres.notification.Repository.GetByID"

	rawModel, err := r.queries(ctx).GetNotificationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, mapError(err))
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func (r *Repository) Cancel(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	const op = "postgres.notification.Repository.Cancel"

	rawModel, err := r.queries(ctx).CancelNotification(ctx, id)
	if err == nil {
		return converters.ConvertGenToDomain(&rawModel), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Nothing was updated: the notification is missing or no longer pending.
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if current.Status == vo.Declined {
		return current, nil
	}

	return nil, fmt.Errorf("%s: status %s: %w", op, current.Status, errorx.ErrNotCancellable)
}

func (r *Repository) ClaimDue(
	ctx context.Context,
	dueBefore, staleBefore time.Time,
	limit int32,
) ([]*model.Notification, error) {
	const op = "postgres.notification.Repository.ClaimDue"

	rawModels, err := r.queries(ctx).ClaimDueNotifications(ctx, gen.ClaimDueNotificationsParams{
		DueBefore:   dueBefore,
		StaleBefore: staleBefore,
		BatchSize:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converters.ConvertGenSliceToDomain(rawModels), nil
}

func (r *Repository) MarkQueued(ctx context.Context, ids []uuid.UUID) error {
	const op = "postgres.notification.Repository.MarkQueued"

	if err := r.queries(ctx).MarkNotificationsQueued(ctx, ids); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *Repository) LockPending(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.one(ctx, "postgres.notification.Repository.LockPending", id, (*gen.Queries).LockPendingNotification)
}

func (r *Repository) MarkSent(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.one(ctx, "postgres.notification.Repository.MarkSent", id, (*gen.Queries).MarkNotificationSent)
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.one(ctx, "postgres.notification.Repository.MarkFailed", id, (*gen.Queries).MarkNotificationFailed)
}

func (r *Repository) MarkRetry(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return r.one(ctx, "postgres.notification.Repository.MarkRetry", id, (*gen.Queries).MarkNotificationRetry)
}

// one runs a query that takes an id and returns a single notification.
func (r *Repository) one(
	ctx context.Context,
	op string,
	id uuid.UUID,
	query func(*gen.Queries, context.Context, uuid.UUID) (gen.Notification, error),
) (*model.Notification, error) {
	rawModel, err := query(r.queries(ctx), ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, mapError(err))
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errorx.ErrNotFound
	}
	return err
}

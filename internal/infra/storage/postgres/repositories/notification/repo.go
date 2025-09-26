package notification

import (
	"context"
	"database/sql"
	"fmt"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/params"
	"wb-tech-l3/internal/domain/core/notification/vo"
	"wb-tech-l3/internal/infra/storage/postgres/repositories/notification/converters"
	"wb-tech-l3/internal/infra/storage/postgres/repositories/notification/gen"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"
)

type Repository struct {
	log      appPorts.Logger
	executor *dbpg.DB
	queries  *gen.Queries
}

func NewRepository(log appPorts.Logger, executor *dbpg.DB) *Repository {
	return &Repository{
		log:      log,
		executor: executor,
		queries:  gen.New(executor.Master),
	}
}

func (r *Repository) Create(ctx context.Context, p params.CreateNotificationParams) (*model.Notification, error) {
	const op = "postgres.notification.Repository.Create"
	parameters := converters.ConvertCreateParams(p)
	rawModel, err := r.queries.CreateNotification(ctx, parameters)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func (r *Repository) Cancel(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	const op = "postgres.notification.Repository.Cancel"

	rawModel, err := r.queries.CancelNotification(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	const op = "postgres.notification.Repository.GetByID"

	rawModel, err := r.queries.GetNotificationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func (r *Repository) UpdateStatus(ctx context.Context, p params.UpdateNotificationStatusParams) (*model.Notification, error) {
	const op = "postgres.notification.Repository.UpdateStatus"

	parameters := converters.ConvertUpdateParams(p)
	rawModel, err := r.queries.UpdateNotificationStatus(ctx, parameters)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return converters.ConvertGenToDomain(&rawModel), nil
}

func (r *Repository) ProcessPending(
	ctx context.Context,
	batchSize int32,
	processor func(ctx context.Context, m *model.Notification) error,
) error {
	const op = "postgres.notification.Repository.ProcessPending"

	tx, err := r.executor.Master.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := r.queries.WithTx(tx)
	notifications, err := qtx.GetPendingNotificationsForUpdate(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("%s: get pending: %w", op, err)
	}

	if len(notifications) == 0 {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.log.Error("failed to rollback empty transaction", "error", rollbackErr)
		}
		return nil
	}

	successIDs := make([]uuid.UUID, 0, len(notifications))
	failedIDs := make([]uuid.UUID, 0)

	for _, n := range notifications {
		notification := converters.ConvertGenToDomain(&n)

		if err = processor(ctx, notification); err != nil {
			r.log.Error("failed to process notification",
				"notification_id", notification.ID,
				"error", err.Error(),
			)
			failedIDs = append(failedIDs, notification.ID)
		} else {
			successIDs = append(successIDs, notification.ID)
			notification.Status = vo.Sent
		}
	}

	if len(successIDs) > 0 {
		if err = qtx.SetNotificationStatusSentMany(ctx, successIDs); err != nil {
			return fmt.Errorf("%s: update sent status: %w", op, err)
		}
	}

	if len(failedIDs) > 0 {
		if err = qtx.SetNotificationStatusFailedMany(ctx, failedIDs); err != nil {
			return fmt.Errorf("%s: update failed status: %w", op, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit transaction: %w", op, err)
	}

	r.log.Info("processed notifications batch",
		"total", len(notifications),
		"success", len(successIDs),
		"failed", len(failedIDs),
	)

	return nil
}

// ReadPendingForUpdate and SetSentMany is not used in app
func (r *Repository) ReadPendingForUpdate(ctx context.Context) ([]*model.Notification, *sql.Tx, error) {
	const op = "postgres.notification.Repository.ReadPendingForUpdate"

	tx, err := r.executor.Master.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := r.queries.WithTx(tx)
	rows, err := qtx.GetPendingNotificationsForUpdate(ctx, 10)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	notifications := make([]*model.Notification, len(rows))
	for i, n := range rows {
		notifications[i] = converters.ConvertGenToDomain(&n)
	}

	return notifications, tx, nil
}

func (r *Repository) SetSentMany(ctx context.Context, tx *sql.Tx, ids []uuid.UUID) error {
	const op = "postgres.notification.Repository.SetSentMany"

	var err error
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := r.queries.WithTx(tx)
	if err = qtx.SetNotificationStatusSentMany(ctx, ids); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return tx.Commit()
}

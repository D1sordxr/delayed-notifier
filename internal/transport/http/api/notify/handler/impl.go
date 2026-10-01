package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/input"
	"github.com/D1sordxr/delayed-notifier/internal/application/notification/port"
	appPorts "github.com/D1sordxr/delayed-notifier/internal/domain/app/ports"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/model"
)

const internalError = "Internal server error"

type Handlers struct {
	log appPorts.Logger
	uc  port.NotifyUseCase
}

func NewHandlers(log appPorts.Logger, uc port.NotifyUseCase) *Handlers {
	return &Handlers{log: log, uc: uc}
}

func (h Handlers) GetHealth(
	context.Context,
	GetHealthRequestObject,
) (GetHealthResponseObject, error) {
	return GetHealth200JSONResponse{Status: "ok"}, nil
}

func (h Handlers) PostNotify(
	ctx context.Context,
	request PostNotifyRequestObject,
) (PostNotifyResponseObject, error) {
	if request.Body == nil {
		return PostNotify400JSONResponse{
			Error: "Request body is required",
		}, nil
	}

	body := request.Body
	if strings.TrimSpace(body.AuthorId) == "" ||
		strings.TrimSpace(body.Subject) == "" ||
		strings.TrimSpace(body.Message) == "" ||
		body.Channel == "" ||
		body.ScheduledAt.IsZero() {
		return PostNotify400JSONResponse{
			Error: "author_id, subject, message, channel and scheduled_at are required fields",
		}, nil
	}

	notification, err := h.uc.Create(ctx, input.CreateNotifyInput{
		AuthorID:   body.AuthorId,
		Subject:    body.Subject,
		Message:    body.Message,
		Channel:    string(body.Channel),
		EmailTo:    body.EmailTo,
		TelegramID: body.TelegramId,
		SmsTo:      body.SmsTo,
		Scheduled:  body.ScheduledAt,
	})
	switch {
	case err == nil:
		return PostNotify201JSONResponse(toResponse(notification)), nil
	case errors.Is(err, errorx.ErrInvalidChannel), errors.Is(err, errorx.ErrInvalidRecipient):
		return PostNotify400JSONResponse{Error: err.Error()}, nil
	default:
		h.log.Error("Failed to create notification", "error", err.Error())
		return PostNotify500JSONResponse{Error: internalError}, nil
	}
}

func (h Handlers) GetNotifyId(
	ctx context.Context,
	request GetNotifyIdRequestObject,
) (GetNotifyIdResponseObject, error) {
	notification, err := h.uc.Read(ctx, request.Id)
	switch {
	case err == nil:
		return GetNotifyId200JSONResponse(toResponse(notification)), nil
	case errors.Is(err, errorx.ErrInvalidID):
		return GetNotifyId400JSONResponse{Error: errorx.ErrInvalidID.Error()}, nil
	case errors.Is(err, errorx.ErrNotFound):
		return GetNotifyId404JSONResponse{Error: errorx.ErrNotFound.Error()}, nil
	default:
		h.log.Error("Failed to read notification", "notification_id", request.Id, "error", err.Error())
		return GetNotifyId500JSONResponse{Error: internalError}, nil
	}
}

func (h Handlers) DeleteNotifyId(
	ctx context.Context,
	request DeleteNotifyIdRequestObject,
) (DeleteNotifyIdResponseObject, error) {
	_, err := h.uc.Cancel(ctx, request.Id)
	switch {
	case err == nil:
		return DeleteNotifyId200JSONResponse{Result: "Notification cancelled successfully"}, nil
	case errors.Is(err, errorx.ErrInvalidID):
		return DeleteNotifyId400JSONResponse{Error: errorx.ErrInvalidID.Error()}, nil
	case errors.Is(err, errorx.ErrNotFound):
		return DeleteNotifyId404JSONResponse{Error: errorx.ErrNotFound.Error()}, nil
	case errors.Is(err, errorx.ErrNotCancellable):
		return DeleteNotifyId409JSONResponse{Error: errorx.ErrNotCancellable.Error()}, nil
	default:
		h.log.Error("Failed to cancel notification", "notification_id", request.Id, "error", err.Error())
		return DeleteNotifyId500JSONResponse{Error: internalError}, nil
	}
}

func toResponse(notification *model.Notification) NotificationResponse {
	id := notification.ID.String()
	channel := NotificationResponseChannel(notification.Channel.String())
	status := NotificationResponseStatus(notification.Status.String())

	return NotificationResponse{
		Id:             &id,
		Subject:        &notification.Subject,
		Message:        &notification.Message,
		AuthorId:       notification.AuthorID,
		EmailTo:        notification.EmailTo,
		TelegramChatId: notification.TelegramChatID,
		SmsTo:          notification.SmsTo,
		Channel:        &channel,
		Status:         &status,
		Attempts:       &notification.Attempts,
		ScheduledAt:    &notification.ScheduledAt,
		SentAt:         notification.SentAt,
		CreatedAt:      &notification.CreatedAt,
		UpdatedAt:      &notification.UpdatedAt,
	}
}

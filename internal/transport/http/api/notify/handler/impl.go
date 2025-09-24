package handler

import (
	"context"
	"fmt"
	"wb-tech-l3/internal/application/notification/input"
	"wb-tech-l3/internal/application/notification/port"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/pkg/xstrings"
)

type Handlers struct {
	uc port.NotifyUseCase
}

func NewHandlers(uc port.NotifyUseCase) *Handlers {
	return &Handlers{uc: uc}
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

	strChannel := string(request.Body.Channel)
	if xstrings.IsEqual(
		"",
		request.Body.Message,
		request.Body.Subject,
		strChannel,
	) || request.Body.ScheduledAt.IsZero() {
		return PostNotify400JSONResponse{
			Error: "user_id, subject, message, channel and scheduled_at are required fields",
		}, nil
	}

	notification, err := h.uc.Create(ctx, input.CreateNotifyInput{
		AuthorID:   request.Body.AuthorId,
		Subject:    request.Body.Subject,
		Message:    request.Body.Message,
		Channel:    strChannel,
		EmailTo:    request.Body.EmailTo,
		TelegramID: request.Body.TelegramId,
		SmsTo:      request.Body.SmsTo,
		Scheduled:  request.Body.ScheduledAt,
	})
	if err != nil {
		return PostNotify500JSONResponse{
			Error: "Failed to create notification: " + err.Error(),
		}, nil
	}

	response := h.parseNotificationResponse(notification)
	return PostNotify201JSONResponse(response), nil
}

func (h Handlers) DeleteNotifyId(
	ctx context.Context,
	request DeleteNotifyIdRequestObject,
) (DeleteNotifyIdResponseObject, error) {
	if request.Id == "" {
		return DeleteNotifyId404JSONResponse{
			Error: "ID is required",
		}, nil
	}

	if err := h.uc.Delete(ctx, request.Id); err != nil {
		return DeleteNotifyId500JSONResponse{
			Error: fmt.Sprintf("Failed to delete notification: %s", err.Error()),
		}, nil
	}

	return DeleteNotifyId200JSONResponse{
		Result: "Notification cancelled successfully",
	}, nil
}

func (h Handlers) GetNotifyId(
	ctx context.Context,
	request GetNotifyIdRequestObject,
) (GetNotifyIdResponseObject, error) {
	if request.Id == "" {
		return GetNotifyId404JSONResponse{
			Error: "Notification ID is required",
		}, nil
	}

	notification, err := h.uc.Read(ctx, request.Id)
	if err != nil {
		return GetNotifyId404JSONResponse{
			Error: "Notification not found: " + err.Error(),
		}, nil
	}

	response := h.parseNotificationResponse(notification)
	return GetNotifyId200JSONResponse(response), nil
}

func (h Handlers) parseNotificationResponse(notification *model.Notification) NotificationResponse {
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

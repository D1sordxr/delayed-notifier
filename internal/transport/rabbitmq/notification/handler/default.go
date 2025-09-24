package handler

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"
	ports2 "wb-tech-l3/internal/domain/core/notification/ports"
	"wb-tech-l3/internal/domain/core/notification/vo"
)

type DefaultDelivery struct {
	log  appPorts.Logger
	pipe ports2.MessagePipe
	cs   ports2.CacheStore
}

func NewDefaultDelivery(
	log appPorts.Logger,
	pipe ports2.MessagePipe,
	cs ports2.CacheStore,
) *DefaultDelivery {
	return &DefaultDelivery{
		log:  log,
		pipe: pipe,
		cs:   cs,
	}
}

func (d *DefaultDelivery) isValidToProcess(ctx context.Context, id string) bool {
	isDeleted, err := d.cs.IsDeleted(ctx, id)
	if err != nil {
		d.log.Warn("Failed to check if notification is deleted", "error", err.Error())
		return false
	}
	return !isDeleted
}

func (d *DefaultDelivery) handleMessage(ctx context.Context, notificationRaw []byte) {
	var notification model.Notification
	if err := json.Unmarshal(notificationRaw, &notification); err != nil {
		d.log.Error("Failed to parse notification", "error", err.Error())
		return
	}

	switch notification.Channel {
	case vo.Email:
		if d.isValidToProcess(ctx, notification.ID.String()) {
			d.log.Info("Sending email")
		}
	case vo.Telegram:
		if d.isValidToProcess(ctx, notification.ID.String()) {
			d.log.Info("Sending telegram")
		}
	case vo.SMS:
		if d.isValidToProcess(ctx, notification.ID.String()) {
			d.log.Info("Sending SMS")
		}
	}
}

func (d *DefaultDelivery) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-d.pipe.GetMessageChan():
			if !ok {
				return nil
			}
			d.handleMessage(ctx, msg)
		}
	}
}

const pipeDrainerCount = 5

func (d *DefaultDelivery) Stop(ctx context.Context) error {
	drainCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var (
		wg   sync.WaitGroup
		done = make(chan struct{})
	)

	for i := 0; i < pipeDrainerCount; i++ {
		wg.Go(func() {
			for {
				select {
				case <-drainCtx.Done():
					return
				case msg, ok := <-d.pipe.GetMessageChan():
					if !ok {
						return
					}
					d.handleMessage(ctx, msg)
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.log.Info("Pipe drained successfully")
	case <-drainCtx.Done():
		d.log.Warn("Pipe draining timeout")
	}

	return nil
}

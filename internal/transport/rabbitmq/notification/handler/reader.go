package handler

import (
	"context"
	"sync"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/ports"
)

type NotificationReader struct {
	log  appPorts.Logger
	pipe ports.MessagePipe

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewNotificationReader(
	log appPorts.Logger,
	pipe ports.MessagePipe,
) *NotificationReader {
	return &NotificationReader{
		log:  log,
		pipe: pipe,
	}
}

func (r *NotificationReader) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.wg.Go(func() {
		for {
			select {
			case <-workerCtx.Done():
				return
			case notification, ok := <-r.pipe.GetMessageChan():
				if !ok {
					return
				}

			}
		}
	})

}

func (r *NotificationReader) Stop(ctx context.Context) error {}

package handler

import (
	"context"
	appPorts "wb-tech-l3/internal/domain/app/ports"
	"wb-tech-l3/internal/domain/core/notification/model"
	"wb-tech-l3/internal/domain/core/notification/ports"
	"wb-tech-l3/internal/domain/core/notification/vo"
)

type Processor struct {
	log      appPorts.Logger
	consumer ports.Consumer
	cancel   context.CancelFunc
}

func NewProcessor(log appPorts.Logger, consumer ports.Consumer) *Processor {
	return &Processor{
		log:      log,
		consumer: consumer,
	}
}

func (p *Processor) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	return p.consumer.StartConsuming(
		workerCtx,
		func(ctx context.Context, m *model.Notification) error {
			switch m.Channel {
			case vo.Email:
				p.log.Info("email received", "data", m)
				// sending email logic
			case vo.Telegram:
				p.log.Info("telegram received", "data", m)
				// sending telegram logic
			case vo.SMS:
				p.log.Info("sms received", "data", m)
				// sending sms logic
			default:
				p.log.Warn("Received message with invalid channel",
					"id", m.ID.String(),
					"channel", m.Channel,
				)
			}

			return nil
		},
	)
}

func (p *Processor) Stop(_ context.Context) error {
	p.cancel()
	return nil
}

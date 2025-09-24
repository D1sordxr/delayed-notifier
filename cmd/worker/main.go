package main

import (
	"context"
	"github.com/wb-go/wbf/zlog"
	"log/slog"
	"os"
	"os/signal"
	"time"
	"wb-tech-l3/internal/infra/logger"

	"wb-tech-l3/internal/application/notification/usecase"
	loadApp "wb-tech-l3/internal/infra/app"
	rabbitAdapter "wb-tech-l3/internal/infra/broker/rabbitmq/notification"
	"wb-tech-l3/internal/infra/cache/redis/notification"
	"wb-tech-l3/internal/infra/config"
	"wb-tech-l3/internal/infra/worker"
	"wb-tech-l3/internal/transport/http"
	"wb-tech-l3/internal/transport/http/api/notify"
	httpHandler "wb-tech-l3/internal/transport/http/api/notify/handler"
	workerHandler "wb-tech-l3/internal/transport/rabbitmq/notification/handler"

	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := config.NewConfig()

	zlog.Init()
	log := logger.New(zlog.Logger)

	log.Debug(
		"Attempting to connect to RabbitMQ",
		"conn_str", cfg.Broker.GetConnectionString(),
	)
	brokerConn, err := rabbitmq.Connect(cfg.Broker.GetConnectionString(), 3, time.Second*3)
	if err != nil {
		log.Error("Broker connection failure")
	}
	notificationSender, err := rabbitAdapter.NewSender(log, brokerConn)
	if err != nil {
		log.Error("Failed to create notification sender", "error", err.Error())
	}
	notificationReader, err := rabbitAdapter.NewReader(log, brokerConn)
	if err != nil {
		log.Error("Failed to create notification reader", "error", err.Error())
	}

	notificationProcessor := workerHandler.NewDefaultDelivery(
		log,
		notificationReader,
		notificationCacheAdapter, // TODO another repo
	)
	notificationWorker := worker.NewWorker(
		log,
		notificationProcessor,
	)

	app := loadApp.NewApp(
		log,
		notificationWorker,
		notificationReader,
		notificationSender,
	)
	app.Run(ctx)
}

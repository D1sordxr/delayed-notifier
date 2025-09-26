package main

import (
	"context"
	"os"
	"os/signal"
	"time"
	"wb-tech-l3/internal/infra/logger"
	"wb-tech-l3/internal/infra/storage/postgres"

	"github.com/rs/zerolog"

	loadApp "wb-tech-l3/internal/infra/app"
	"wb-tech-l3/internal/infra/config"
	"wb-tech-l3/internal/infra/worker"

	rabbitAdapter "wb-tech-l3/internal/infra/broker/rabbitmq/notification"
	"wb-tech-l3/internal/infra/cache/redis/notification"
	notificationRepository "wb-tech-l3/internal/infra/storage/postgres/repositories/notification"
	workerHandler "wb-tech-l3/internal/transport/rabbitmq/notification/handler"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := config.NewWorkerConfig()

	log := logger.New(defaultLogger)
	log.Debug("Config data", "config", cfg)

	storageConn, err := dbpg.New(cfg.Storage.ConnectionString(), nil, nil)
	if err != nil {
		log.Error("Failed to connect to database", "error", err.Error())
		return
	}
	defer func() { _ = storageConn.Master.Close() }()
	if err = postgres.SetupStorage(storageConn.Master, cfg.Storage); err != nil {
		log.Error("Failed to setup storage", "error", err.Error())
		return
	}
	notificationRepo := notificationRepository.NewRepository(log, storageConn)

	cacheConn := redis.New(cfg.Cache.ClientAddress, cfg.Cache.Password, 1)
	defer func() { _ = cacheConn.Close() }()
	notificationCacheAdapter := notification.NewAdapter(cacheConn)

	brokerConn, err := rabbitmq.Connect(cfg.Broker.GetConnectionString(), 3, time.Second*3)
	if err != nil {
		log.Error("Broker connection failure", "error", err.Error())
	}
	defer func() { _ = brokerConn.Close() }()

	notificationQueue, err := rabbitAdapter.NewQueue(log, brokerConn, cfg.Broker.DeclareExchange)
	if err != nil {
		log.Error("RabbitMQ queue failure", "error", err.Error())
		return
	}
	notificationProducer := rabbitAdapter.NewProducer(notificationQueue)
	notificationConsumer := rabbitAdapter.NewConsumer(log, notificationQueue)

	notificationWriter := workerHandler.NewNotificationWriter(
		log,
		notificationRepo,
		notificationProducer,
		notificationCacheAdapter,
	)
	notificationProcessor := workerHandler.NewProcessor(log, notificationConsumer)
	notificationWorker := worker.NewWorker(
		log,
		notificationProcessor,
		notificationWriter,
	)

	app := loadApp.NewApp(
		log,
		notificationWorker,
	)
	app.Run(ctx)
}

var defaultLogger zerolog.Logger

func init() {
	zlog.Init()
	defaultLogger = zlog.Logger
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/D1sordxr/delayed-notifier/internal/application/notification/dispatcher"
	"github.com/D1sordxr/delayed-notifier/internal/application/notification/scheduler"
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/vo"
	broker "github.com/D1sordxr/delayed-notifier/internal/infra/broker/rabbitmq/notification"
	notificationCache "github.com/D1sordxr/delayed-notifier/internal/infra/cache/redis/notification"
	"github.com/D1sordxr/delayed-notifier/internal/infra/config"
	"github.com/D1sordxr/delayed-notifier/internal/infra/logger"
	"github.com/D1sordxr/delayed-notifier/internal/infra/sender"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres"
	notificationRepository "github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification"
	"github.com/D1sordxr/delayed-notifier/internal/infra/worker"
	workerHandler "github.com/D1sordxr/delayed-notifier/internal/transport/rabbitmq/notification/handler"

	"github.com/D1sordxr/packages/app"
	"github.com/D1sordxr/packages/cron"
	pgPool "github.com/D1sordxr/packages/postgres"
	exec "github.com/D1sordxr/packages/postgres/executor"
	"github.com/D1sordxr/packages/postgres/tx"
	"github.com/D1sordxr/packages/rabbitmq"
	"github.com/D1sordxr/packages/redis"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.NewWorkerConfig()
	log := logger.New(cfg.LogLevel)

	pool, err := pgPool.NewPool(ctx, &cfg.Storage.Config)
	if err != nil {
		log.Error("Failed to connect to database", "error", err.Error())
		return err
	}
	poolComponent := pgPool.NewPoolComponent(pool, log, 0)
	defer func() { _ = poolComponent.Shutdown(context.Background()) }()

	if cfg.Storage.Migrations {
		if err = postgres.Migrate(ctx, pool); err != nil {
			log.Error("Failed to apply migrations", "error", err.Error())
			return err
		}
	}
	executor := exec.NewExecutor(pool)
	txManager := tx.NewManager(executor)
	notificationRepo := notificationRepository.NewRepository(executor)

	cacheClient, err := redis.NewClient(ctx, &cfg.Cache.Config)
	if err != nil {
		log.Error("Failed to connect to cache", "error", err.Error())
		return err
	}
	cacheComponent := redis.NewClientComponent(cacheClient, log, 0)
	defer func() { _ = cacheComponent.Shutdown(context.Background()) }()

	cacheWriter := notificationCache.NewWriter(
		log,
		notificationCache.NewStore(cacheClient, cfg.Cache.TTL),
		cfg.Cache.WriteBuffer,
		cfg.Cache.WriteWorkers,
	)

	brokerConn, err := rabbitmq.Dial(ctx, &cfg.Broker)
	if err != nil {
		log.Error("Failed to connect to broker", "error", err.Error())
		return err
	}
	brokerComponent := rabbitmq.NewConnectionComponent(brokerConn)
	defer func() { _ = brokerComponent.Shutdown(context.Background()) }()

	if err = broker.Topology(cfg.Dispatcher.RetryDelay).Declare(brokerConn); err != nil {
		log.Error("Failed to declare broker topology", "error", err.Error())
		return err
	}
	rawPublisher, err := rabbitmq.NewPublisher(brokerConn)
	if err != nil {
		log.Error("Failed to open broker publisher", "error", err.Error())
		return err
	}
	publisher := broker.NewPublisher(rawPublisher)

	schedulerUC := scheduler.NewUseCase(log, txManager, notificationRepo, publisher, scheduler.Config{
		BatchSize:  cfg.Scheduler.BatchSize,
		Lookahead:  cfg.Scheduler.Lookahead,
		StaleAfter: cfg.Scheduler.StaleAfter,
	})

	logSender := sender.NewLog(log)
	dispatcherUC := dispatcher.NewUseCase(
		log,
		txManager,
		notificationRepo,
		publisher,
		sender.Router{
			vo.Email:    logSender,
			vo.Telegram: logSender,
			vo.SMS:      logSender,
		},
		cacheWriter,
		cfg.Dispatcher.MaxAttempts,
	)

	dispatchConsumer := rabbitmq.NewConsumer(
		brokerConn,
		rabbitmq.ConsumerConfig{
			Queue:    broker.NotificationsQueue,
			Prefetch: cfg.Dispatcher.Prefetch,
		},
		workerHandler.NewDispatchHandler(dispatcherUC),
		log,
	)

	schedulerWorker := cron.NewWorker(
		worker.NewScheduler(log, schedulerUC, cfg.Scheduler.Interval, int(cfg.Scheduler.BatchSize)),
	)

	// Components stop in reverse order: the scheduler stops claiming first,
	// the dispatcher finishes in-flight deliveries, the cache writer flushes,
	// and the connections are closed last.
	return app.New(log,
		poolComponent,
		cacheComponent,
		brokerComponent,
		cacheWriter,
		dispatchConsumer,
		schedulerWorker,
	).Run(ctx)
}

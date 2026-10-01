package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	notificationUseCase "github.com/D1sordxr/delayed-notifier/internal/application/notification/usecase"
	notificationCache "github.com/D1sordxr/delayed-notifier/internal/infra/cache/redis/notification"
	"github.com/D1sordxr/delayed-notifier/internal/infra/config"
	"github.com/D1sordxr/delayed-notifier/internal/infra/logger"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres"
	notificationRepository "github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/api/notify"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/api/notify/handler"

	"github.com/D1sordxr/packages/app"
	"github.com/D1sordxr/packages/httpserver"
	pgPool "github.com/D1sordxr/packages/postgres"
	exec "github.com/D1sordxr/packages/postgres/executor"
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

	cfg := config.NewApiConfig()
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
	notificationRepo := notificationRepository.NewRepository(exec.NewExecutor(pool))

	cacheClient, err := redis.NewClient(ctx, &cfg.Cache.Config)
	if err != nil {
		log.Error("Failed to connect to cache", "error", err.Error())
		return err
	}
	cacheComponent := redis.NewClientComponent(cacheClient, log, 0)
	defer func() { _ = cacheComponent.Shutdown(context.Background()) }()

	cacheStore := notificationCache.NewStore(cacheClient, cfg.Cache.TTL)
	cacheWriter := notificationCache.NewWriter(log, cacheStore, cfg.Cache.WriteBuffer, cfg.Cache.WriteWorkers)

	notificationUC := notificationUseCase.NewUseCase(
		log,
		cacheStore,
		cacheWriter,
		notificationRepo,
	)

	httpServer := httpserver.New(cfg.Server.Config, http.NewHandler(
		&cfg.Server,
		notify.NewRouteRegisterer(handler.NewHandlers(log, notificationUC)),
	))

	// Components stop in reverse order: the server stops taking requests
	// first, the connections it uses are closed last.
	return app.New(log,
		poolComponent,
		cacheComponent,
		cacheWriter,
		httpServer,
	).Run(ctx)
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/D1sordxr/delayed-notifier/internal/infra/logger"
	"github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres"

	loadApp "github.com/D1sordxr/delayed-notifier/internal/infra/app"
	"github.com/D1sordxr/delayed-notifier/internal/infra/config"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http"

	notificationUseCase "github.com/D1sordxr/delayed-notifier/internal/application/notification/usecase"
	notificationCache "github.com/D1sordxr/delayed-notifier/internal/infra/cache/redis/notification"
	notificationRepository "github.com/D1sordxr/delayed-notifier/internal/infra/storage/postgres/repositories/notification"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/api/notify"
	"github.com/D1sordxr/delayed-notifier/internal/transport/http/api/notify/handler"

	"github.com/rs/zerolog"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.NewApiConfig()

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
	notificationCacheAdapter := notificationCache.NewAdapter(cacheConn)

	notificationUC := notificationUseCase.NewUseCase(
		log,
		notificationCacheAdapter,
		notificationRepo,
	)

	notificationHandlers := handler.NewHandlers(notificationUC)
	notificationRouteRegisterer := notify.NewRouteRegisterer(notificationHandlers)

	httpServer := http.NewServer(
		log,
		&cfg.Server,
		notificationRouteRegisterer,
	)

	app := loadApp.NewApp(
		log,
		httpServer,
	)
	app.Run(ctx)
}

var defaultLogger zerolog.Logger

func init() {
	zlog.Init()
	defaultLogger = zlog.Logger
}

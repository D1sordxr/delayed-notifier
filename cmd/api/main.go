package main

import (
	"context"
	"github.com/rs/zerolog"
	"os"
	"os/signal"
	"syscall"
	"wb-tech-l3/internal/infra/logger"

	loadApp "wb-tech-l3/internal/infra/app"
	"wb-tech-l3/internal/infra/config"
	"wb-tech-l3/internal/transport/http"

	notificationUseCase "wb-tech-l3/internal/application/notification/usecase"
	notificationCache "wb-tech-l3/internal/infra/cache/redis/notification"
	notificationRepository "wb-tech-l3/internal/infra/storage/postgres/repositories/notification"
	"wb-tech-l3/internal/transport/http/api/notify"
	"wb-tech-l3/internal/transport/http/api/notify/handler"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/zlog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.NewConfig()

	log := logger.New(defaultLogger)

	storageConn, err := dbpg.New(cfg.Storage.ConnectionString(), nil, nil)
	if err != nil {
		panic("Failed to connect to database")
	}
	defer func() { _ = storageConn.Master.Close() }()
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

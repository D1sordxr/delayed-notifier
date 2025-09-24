package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wb-tech-l3/internal/domain/app/ports"

	"golang.org/x/sync/errgroup"
)

type component interface {
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type App struct {
	log        ports.Logger
	components []component
}

func NewApp(
	log ports.Logger,
	components ...component,
) *App {
	return &App{
		log:        log,
		components: components,
	}
}

func (a *App) Run(ctx context.Context) {
	defer a.shutdown()

	errChan := make(chan error)
	errGroup, ctx := errgroup.WithContext(ctx)
	go func() { errChan <- errGroup.Wait() }()

	for i, comp := range a.components {
		c := comp
		idx := i
		errGroup.Go(func() error {
			a.log.Info("Starting component", "idx", idx, "type", fmt.Sprintf("%T", c))
			err := c.Run(ctx)
			if err != nil {
				a.log.Error("Component failed", "idx", idx, "type", fmt.Sprintf("%T", c), "error", err.Error())
			} else {
				a.log.Info("Component stopped", "idx", idx, "type", fmt.Sprintf("%T", c))
			}
			return err
		})
	}

	select {
	case err := <-errChan:
		a.log.Error("App received an error", "error", err.Error())
	case <-ctx.Done():
		a.log.Info("App received a terminate signal")
	}
}

func (a *App) shutdown() {
	a.log.Info("App shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	errs := make([]error, 0, len(a.components))
	for i := len(a.components) - 1; i >= 0; i-- {
		a.log.Info("Shutting down component", "idx", i)
		if err := a.components[i].Shutdown(shutdownCtx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		a.log.Info("App successfully shutdown")
	} else {
		a.log.Error(
			"App shutdown with errors",
			"errors", errors.Join(errs...).Error(),
		)
	}
}

package postgres

import (
	"context"
	"embed"
	"fmt"
	"time"

	"wb-tech-l3/internal/infra/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Pool struct {
	*pgxpool.Pool
	migrations bool
}

func NewPool(ctx context.Context, config *config.Postgres) *Pool {
	pool, err := pgxpool.New(ctx, config.ConnectionString())
	if err != nil {
		panic(err)
	}

	return &Pool{
		Pool:       pool,
		migrations: config.Migrations,
	}
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func (p *Pool) migrate() error {
	sqlDB := stdlib.OpenDBFromPool(p.Pool)
	defer func() { _ = sqlDB.Close() }()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return err
	}

	return nil
}

func (p *Pool) Run(ctx context.Context) error {
	const op = "storage.Pool.Run"

	if p.migrations {
		if err := p.migrate(); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.Pool.Ping(ctx); err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}
	}
}

func (p *Pool) Shutdown(_ context.Context) error {
	p.Pool.Close()
	return nil
}

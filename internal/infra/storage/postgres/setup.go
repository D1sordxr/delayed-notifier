package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// Migrate applies the embedded goose migrations. A Postgres advisory lock
// makes it safe for several services or replicas to migrate on startup.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	const op = "postgres.Migrate"

	migrations, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

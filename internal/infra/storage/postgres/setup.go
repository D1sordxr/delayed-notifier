package postgres

import (
	"database/sql"
	"wb-tech-l3/internal/infra/config"

	"github.com/pressly/goose/v3"
)

func SetupStorage(db *sql.DB, cfg config.Postgres) error {
	if cfg.Migrations {
		goose.SetBaseFS(embedMigrations)
		if err := goose.SetDialect("postgres"); err != nil {
			return err
		}
		if err := goose.Up(db, "migrations"); err != nil {
			return err
		}
	}
	return nil
}

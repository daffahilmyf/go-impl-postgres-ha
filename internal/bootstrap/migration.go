package bootstrap

import (
	"context"
	"database/sql"
	"errors"

	"github.com/daffahilmyf/go-impl-postgres-ha/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

func Migrate(ctx context.Context, cfg config.Config, cmd string, version int64) error {
	if cfg.Database.WriteDSN == "" {
		return errors.New("db: WriteDSN is required")
	}

	pgxCfg, err := pgx.ParseConfig(cfg.Database.WriteDSN)
	if err != nil {
		return err
	}
	pgxCfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	var db *sql.DB
	db = stdlib.OpenDB(*pgxCfg)
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	actions := map[string]func() error{
		"up":      func() error { return goose.Up(db, migrationsDir) },
		"down":    func() error { return goose.Down(db, migrationsDir) },
		"status":  func() error { return goose.Status(db, migrationsDir) },
		"version": func() error { return goose.Version(db, migrationsDir) },
		"redo":    func() error { return goose.Redo(db, migrationsDir) },
		"reset":   func() error { return goose.Reset(db, migrationsDir) },
		"up-to":   func() error { return goose.UpTo(db, migrationsDir, version) },
		"down-to": func() error { return goose.DownTo(db, migrationsDir, version) },
	}
	action, ok := actions[cmd]
	if !ok {
		return errors.New("unknown migrate command")
	}
	return action()
}

package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var fs embed.FS

type Option func(*config) error

func Migrate(ctx context.Context, db *sql.DB, options ...Option) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("db migrations failed: %w", err)
		}
	}()
	cfg := &config{
		Context:         ctx,
		MigrationsTable: DefaultMigrationsTable,
	}

	for _, opt := range options {
		err = opt(cfg)
		if err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}

	driver, err := withInstance(db, cfg)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}
	d, err := iofs.New(fs, "sql")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		"sqlite",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	v, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	if !dirty {
		err = m.Up()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		return nil
	}

	err = m.Force(int(v) - 1)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		return err
	}

	return nil
}

func WithMigrationTableName(name string) Option {
	return func(cfg *config) error {
		if len(name) == 0 {
			return fmt.Errorf("migration table name cannot be empty")
		}
		cfg.MigrationsTable = name
		return nil
	}
}

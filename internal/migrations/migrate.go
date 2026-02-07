package migrations

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	migrationfiles "github.com/onlyspans/events/migrations"
)

// Run executes database migrations embedded at build time.
func Run(databaseURL string) error {
	slog.Info("starting database migrations")

	source, err := iofs.New(migrationfiles.FS, ".")
	if err != nil {
		return fmt.Errorf("create migrations source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("no migrations to apply, database is up to date")
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}

	slog.Info("migrations completed successfully")
	return nil
}

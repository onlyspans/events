package migrations

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Run executes database migrations from the migrations directory.
// It uses golang-migrate's standard approach with file-based migrations.
func Run(databaseURL string) error {
	slog.Info("starting database migrations")

	m, err := migrate.New("file://migrations", databaseURL)
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

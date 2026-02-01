package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/onlyspans/events/internal/domain"
)

// SettingsRepository handles settings data access operations.
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new SettingsRepository.
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves the global settings.
func (r *SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	settings := &domain.Settings{}

	err := r.db.QueryRowContext(ctx, `
		SELECT id, retention_period_days, updated_at, updated_by
		FROM settings
		WHERE id = $1
	`, domain.GlobalSettingsID).Scan(
		&settings.ID,
		&settings.RetentionPeriodDays,
		&settings.UpdatedAt,
		&settings.UpdatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

// Save saves or updates the settings.
func (r *SettingsRepository) Save(ctx context.Context, settings *domain.Settings) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (id, retention_period_days, updated_at, updated_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			retention_period_days = EXCLUDED.retention_period_days,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`,
		settings.ID,
		settings.RetentionPeriodDays,
		settings.UpdatedAt,
		settings.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

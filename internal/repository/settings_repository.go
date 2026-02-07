package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/ports"
)

// Compile-time check that SettingsRepository implements ports.SettingsRepository.
var _ ports.SettingsRepository = (*SettingsRepository)(nil)

// SettingsRepository handles settings data access operations.
type SettingsRepository struct {
	pool *pgxpool.Pool
}

// NewSettingsRepository creates a new SettingsRepository.
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

// Get retrieves the global settings.
func (r *SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	settings := &domain.Settings{}

	err := r.pool.QueryRow(ctx, `
		SELECT id, retention_period_days, max_export_size, updated_at, updated_by
		FROM settings
		WHERE id = $1
	`, domain.GlobalSettingsID).Scan(
		&settings.ID,
		&settings.RetentionPeriodDays,
		&settings.MaxExportSize,
		&settings.UpdatedAt,
		&settings.UpdatedBy,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return settings, nil
}

// Save saves or updates the settings.
func (r *SettingsRepository) Save(ctx context.Context, settings *domain.Settings) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO settings (id, retention_period_days, max_export_size, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			retention_period_days = EXCLUDED.retention_period_days,
			max_export_size = EXCLUDED.max_export_size,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`,
		settings.ID,
		settings.RetentionPeriodDays,
		settings.MaxExportSize,
		settings.UpdatedAt,
		settings.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/dto"
	"github.com/onlyspans/events/internal/repository"
)

// SettingsService handles settings business logic.
type SettingsService struct {
	repo                 *repository.SettingsRepository
	defaultRetentionDays int
	defaultMaxExportSize int
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(repo *repository.SettingsRepository, defaultRetentionDays, defaultMaxExportSize int) *SettingsService {
	return &SettingsService{
		repo:                 repo,
		defaultRetentionDays: defaultRetentionDays,
		defaultMaxExportSize: defaultMaxExportSize,
	}
}

// GetSettings retrieves the current settings.
func (s *SettingsService) GetSettings(ctx context.Context) (*dto.SettingsDTO, error) {
	settings, err := s.repo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	retentionDays := s.defaultRetentionDays
	if settings != nil {
		retentionDays = settings.RetentionPeriodDays
	}

	return &dto.SettingsDTO{
		RetentionPeriodDays: retentionDays,
		MaxExportSize:       s.defaultMaxExportSize,
	}, nil
}

// UpdateSettings updates the settings.
func (s *SettingsService) UpdateSettings(ctx context.Context, settingsDTO *dto.SettingsDTO) (*dto.SettingsDTO, error) {
	// Validate
	if settingsDTO.RetentionPeriodDays < 1 || settingsDTO.RetentionPeriodDays > 3650 {
		return nil, fmt.Errorf("retention period must be between 1 and 3650 days")
	}

	settings := &domain.Settings{
		ID:                  domain.GlobalSettingsID,
		RetentionPeriodDays: settingsDTO.RetentionPeriodDays,
		UpdatedAt:           time.Now(),
		UpdatedBy:           "api-user",
	}

	if err := s.repo.Save(ctx, settings); err != nil {
		return nil, fmt.Errorf("failed to save settings: %w", err)
	}

	return settingsDTO, nil
}

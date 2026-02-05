package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/onlyspans/events/internal/ports"
	"github.com/robfig/cron/v3"
)

// RetentionService handles scheduled deletion of old events.
type RetentionService struct {
	eventRepo       ports.EventRepository
	settingsService *SettingsService
	cron            *cron.Cron
	logger          *slog.Logger
}

// NewRetentionService creates a new RetentionService.
func NewRetentionService(
	eventRepo ports.EventRepository,
	settingsService *SettingsService,
	logger *slog.Logger,
) *RetentionService {
	return &RetentionService{
		eventRepo:       eventRepo,
		settingsService: settingsService,
		cron:            cron.New(),
		logger:          logger,
	}
}

// Start begins the scheduled retention job.
func (s *RetentionService) Start(cronSpec string) error {
	_, err := s.cron.AddFunc(cronSpec, func() {
		if err := s.ApplyRetention(context.Background()); err != nil {
			s.logger.Error("retention job failed", "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to schedule retention job: %w", err)
	}

	s.cron.Start()
	s.logger.Info("retention service started", "cron", cronSpec)
	return nil
}

// Stop stops the retention service.
func (s *RetentionService) Stop() {
	if s.cron != nil {
		s.cron.Stop()
		s.logger.Info("retention service stopped")
	}
}

// ApplyRetention deletes events older than the configured retention period.
func (s *RetentionService) ApplyRetention(ctx context.Context) error {
	settings, err := s.settingsService.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	retentionDays := settings.RetentionPeriodDays
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	s.logger.Info("applying retention policy",
		"retentionDays", retentionDays,
		"cutoffDate", cutoffDate)

	deletedCount, err := s.eventRepo.DeleteOlderThan(ctx, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete old events: %w", err)
	}

	s.logger.Info("retention policy applied",
		"deletedCount", deletedCount,
		"cutoffDate", cutoffDate)

	return nil
}

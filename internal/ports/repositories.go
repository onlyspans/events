package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) (uuid.UUID, error)
	SaveBatch(ctx context.Context, events []*domain.Event) error
	Search(ctx context.Context, query EventSearchQuery) ([]*domain.Event, int64, error)
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
}

type EventSearchQuery struct {
	EntityID   string
	EntityName string
	Action     string
	UserID     string
	Tenant     string
	StartDate  *time.Time
	EndDate    *time.Time
	SortBy     string
	SortOrder  string
	Page       int
	Size       int
}

type SettingsRepository interface {
	Get(ctx context.Context) (*domain.Settings, error)
	Save(ctx context.Context, settings *domain.Settings) error
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

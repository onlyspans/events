// Package ports defines the interfaces (ports) that decouple the application layers.
// These interfaces follow the ports and adapters (hexagonal) architecture pattern,
// allowing the business logic to remain independent of infrastructure concerns.
package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
)

// EventRepository defines the contract for event data access operations.
// Implementations of this interface handle the persistence of events,
// including creating, searching, and deleting events.
type EventRepository interface {
	// Create inserts a single event into the database and returns its ID.
	Create(ctx context.Context, event *domain.Event) (uuid.UUID, error)

	// SaveBatch saves multiple events in a single transaction for efficiency.
	SaveBatch(ctx context.Context, events []*domain.Event) error

	// Search retrieves events matching the query criteria with pagination.
	// Returns the matching events, total count, and any error.
	Search(ctx context.Context, query EventSearchQuery) ([]*domain.Event, int64, error)

	// DeleteOlderThan deletes events older than the specified cutoff date.
	// Returns the number of deleted events.
	DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
}

// EventSearchQuery represents search criteria for events.
// This type is used by the repository to filter and paginate event queries.
type EventSearchQuery struct {
	User          string
	Category      string
	Action        string
	Document      string
	Project       string
	Environment   string
	Tenant        string
	CorrelationID string
	TraceID       string
	StartDate     *time.Time
	EndDate       *time.Time
	SortBy        string
	SortOrder     string
	Page          int
	Size          int
}

// SettingsRepository defines the contract for settings data access operations.
// Implementations handle the persistence of application settings.
type SettingsRepository interface {
	// Get retrieves the global settings.
	// Returns nil if no settings exist.
	Get(ctx context.Context) (*domain.Settings, error)

	// Save creates or updates the settings.
	Save(ctx context.Context, settings *domain.Settings) error
}

// HealthChecker defines the contract for health check operations.
// This is used by the health handler to verify service dependencies.
type HealthChecker interface {
	// Ping checks if the underlying service is healthy.
	Ping(ctx context.Context) error
}

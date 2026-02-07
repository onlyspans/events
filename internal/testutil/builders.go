package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/dto"
)

// EventBuilder provides a fluent API for building test events
type EventBuilder struct {
	event *domain.Event
}

// NewEventBuilder creates a new EventBuilder with sensible defaults
func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		event: &domain.Event{
			ID:        uuid.New(),
			Timestamp: time.Now().UTC(),
			User:      "test-user",
			Category:  "test-category",
			Action:    "test-action",
			Details:   &domain.EventDetails{},
		},
	}
}

// WithID sets the event ID
func (b *EventBuilder) WithID(id uuid.UUID) *EventBuilder {
	b.event.ID = id
	return b
}

// WithTimestamp sets the event timestamp
func (b *EventBuilder) WithTimestamp(t time.Time) *EventBuilder {
	b.event.Timestamp = t
	return b
}

// WithUser sets the event user
func (b *EventBuilder) WithUser(user string) *EventBuilder {
	b.event.User = user
	return b
}

// WithCategory sets the event category
func (b *EventBuilder) WithCategory(category string) *EventBuilder {
	b.event.Category = category
	return b
}

// WithAction sets the event action
func (b *EventBuilder) WithAction(action string) *EventBuilder {
	b.event.Action = action
	return b
}

// WithDocumentName sets the document name
func (b *EventBuilder) WithDocumentName(name string) *EventBuilder {
	b.event.DocumentName = name
	return b
}

// WithProject sets the project
func (b *EventBuilder) WithProject(project string) *EventBuilder {
	b.event.Project = project
	return b
}

// WithEnvironment sets the environment
func (b *EventBuilder) WithEnvironment(env string) *EventBuilder {
	b.event.Environment = env
	return b
}

// WithTenant sets the tenant
func (b *EventBuilder) WithTenant(tenant string) *EventBuilder {
	b.event.Tenant = tenant
	return b
}

// WithCorrelationID sets the correlation ID
func (b *EventBuilder) WithCorrelationID(id string) *EventBuilder {
	b.event.CorrelationID = id
	return b
}

// WithTraceID sets the trace ID
func (b *EventBuilder) WithTraceID(id string) *EventBuilder {
	b.event.TraceID = id
	return b
}

// WithDetails sets the event details
func (b *EventBuilder) WithDetails(details *domain.EventDetails) *EventBuilder {
	b.event.Details = details
	return b
}

// WithIPAddress sets the IP address in details
func (b *EventBuilder) WithIPAddress(ip string) *EventBuilder {
	if b.event.Details == nil {
		b.event.Details = &domain.EventDetails{}
	}
	b.event.Details.IPAddress = ip
	return b
}

// WithUserAgent sets the user agent in details
func (b *EventBuilder) WithUserAgent(ua string) *EventBuilder {
	if b.event.Details == nil {
		b.event.Details = &domain.EventDetails{}
	}
	b.event.Details.UserAgent = ua
	return b
}

// WithChanges sets the changes in details
func (b *EventBuilder) WithChanges(changes []domain.Change) *EventBuilder {
	if b.event.Details == nil {
		b.event.Details = &domain.EventDetails{}
	}
	b.event.Details.Changes = changes
	return b
}

// WithAdditionalInfo sets additional info in details
func (b *EventBuilder) WithAdditionalInfo(info string) *EventBuilder {
	if b.event.Details == nil {
		b.event.Details = &domain.EventDetails{}
	}
	b.event.Details.AdditionalInfo = info
	return b
}

// Build returns the constructed event
func (b *EventBuilder) Build() *domain.Event {
	return b.event
}

// EventDTOBuilder provides a fluent API for building test event DTOs
type EventDTOBuilder struct {
	dto *dto.EventDTO
}

// NewEventDTOBuilder creates a new EventDTOBuilder with sensible defaults
func NewEventDTOBuilder() *EventDTOBuilder {
	return &EventDTOBuilder{
		dto: &dto.EventDTO{
			Timestamp: time.Now().UTC(),
			User:      "test-user",
			Category:  "test-category",
			Action:    "test-action",
			Details:   &dto.EventDetailsDTO{},
		},
	}
}

// WithTimestamp sets the DTO timestamp
func (b *EventDTOBuilder) WithTimestamp(t time.Time) *EventDTOBuilder {
	b.dto.Timestamp = t
	return b
}

// WithUser sets the DTO user
func (b *EventDTOBuilder) WithUser(user string) *EventDTOBuilder {
	b.dto.User = user
	return b
}

// WithCategory sets the DTO category
func (b *EventDTOBuilder) WithCategory(category string) *EventDTOBuilder {
	b.dto.Category = category
	return b
}

// WithAction sets the DTO action
func (b *EventDTOBuilder) WithAction(action string) *EventDTOBuilder {
	b.dto.Action = action
	return b
}

// WithDocumentName sets the document name
func (b *EventDTOBuilder) WithDocumentName(name string) *EventDTOBuilder {
	b.dto.DocumentName = name
	return b
}

// WithProject sets the project
func (b *EventDTOBuilder) WithProject(project string) *EventDTOBuilder {
	b.dto.Project = project
	return b
}

// WithEnvironment sets the environment
func (b *EventDTOBuilder) WithEnvironment(env string) *EventDTOBuilder {
	b.dto.Environment = env
	return b
}

// WithTenant sets the tenant
func (b *EventDTOBuilder) WithTenant(tenant string) *EventDTOBuilder {
	b.dto.Tenant = tenant
	return b
}

// WithCorrelationID sets the correlation ID
func (b *EventDTOBuilder) WithCorrelationID(id string) *EventDTOBuilder {
	b.dto.CorrelationID = id
	return b
}

// WithTraceID sets the trace ID
func (b *EventDTOBuilder) WithTraceID(id string) *EventDTOBuilder {
	b.dto.TraceID = id
	return b
}

// WithIPAddress sets the IP address in details
func (b *EventDTOBuilder) WithIPAddress(ip string) *EventDTOBuilder {
	if b.dto.Details == nil {
		b.dto.Details = &dto.EventDetailsDTO{}
	}
	b.dto.Details.IPAddress = ip
	return b
}

// WithUserAgent sets the user agent in details
func (b *EventDTOBuilder) WithUserAgent(ua string) *EventDTOBuilder {
	if b.dto.Details == nil {
		b.dto.Details = &dto.EventDetailsDTO{}
	}
	b.dto.Details.UserAgent = ua
	return b
}

// WithChanges sets the changes in details
func (b *EventDTOBuilder) WithChanges(changes []dto.ChangeDTO) *EventDTOBuilder {
	if b.dto.Details == nil {
		b.dto.Details = &dto.EventDetailsDTO{}
	}
	b.dto.Details.Changes = changes
	return b
}

// WithAdditionalInfo sets additional info in details
func (b *EventDTOBuilder) WithAdditionalInfo(info string) *EventDTOBuilder {
	if b.dto.Details == nil {
		b.dto.Details = &dto.EventDetailsDTO{}
	}
	b.dto.Details.AdditionalInfo = info
	return b
}

// Build returns the constructed DTO
func (b *EventDTOBuilder) Build() *dto.EventDTO {
	return b.dto
}

// EventIngestRequestBuilder provides a fluent API for ingest request payloads.
type EventIngestRequestBuilder struct {
	request *dto.EventIngestRequest
}

// NewEventIngestRequestBuilder creates a builder with defaults.
func NewEventIngestRequestBuilder() *EventIngestRequestBuilder {
	return &EventIngestRequestBuilder{
		request: &dto.EventIngestRequest{
			Timestamp: time.Now().UTC(),
			User:      "test-user",
			Category:  "test-category",
			Action:    "test-action",
		},
	}
}

// WithTimestamp sets the request timestamp.
func (b *EventIngestRequestBuilder) WithTimestamp(t time.Time) *EventIngestRequestBuilder {
	b.request.Timestamp = t
	return b
}

// WithUser sets the request user.
func (b *EventIngestRequestBuilder) WithUser(user string) *EventIngestRequestBuilder {
	b.request.User = user
	return b
}

// WithCategory sets the request category.
func (b *EventIngestRequestBuilder) WithCategory(category string) *EventIngestRequestBuilder {
	b.request.Category = category
	return b
}

// WithAction sets the request action.
func (b *EventIngestRequestBuilder) WithAction(action string) *EventIngestRequestBuilder {
	b.request.Action = action
	return b
}

// WithDocumentName sets the document name.
func (b *EventIngestRequestBuilder) WithDocumentName(name string) *EventIngestRequestBuilder {
	b.request.DocumentName = name
	return b
}

// WithProject sets the project.
func (b *EventIngestRequestBuilder) WithProject(project string) *EventIngestRequestBuilder {
	b.request.Project = project
	return b
}

// WithEnvironment sets the environment.
func (b *EventIngestRequestBuilder) WithEnvironment(env string) *EventIngestRequestBuilder {
	b.request.Environment = env
	return b
}

// WithTenant sets the tenant.
func (b *EventIngestRequestBuilder) WithTenant(tenant string) *EventIngestRequestBuilder {
	b.request.Tenant = tenant
	return b
}

// WithDetails sets details.
func (b *EventIngestRequestBuilder) WithDetails(details map[string]interface{}) *EventIngestRequestBuilder {
	b.request.Details = details
	return b
}

// WithCorrelationID sets the correlation ID.
func (b *EventIngestRequestBuilder) WithCorrelationID(id string) *EventIngestRequestBuilder {
	b.request.CorrelationID = id
	return b
}

// WithTraceID sets the trace ID.
func (b *EventIngestRequestBuilder) WithTraceID(id string) *EventIngestRequestBuilder {
	b.request.TraceID = id
	return b
}

// Build returns the constructed ingest request.
func (b *EventIngestRequestBuilder) Build() dto.EventIngestRequest {
	return *b.request
}

// SettingsBuilder provides a fluent API for building test settings
type SettingsBuilder struct {
	settings *domain.Settings
}

// NewSettingsBuilder creates a new SettingsBuilder with sensible defaults
func NewSettingsBuilder() *SettingsBuilder {
	return &SettingsBuilder{
		settings: &domain.Settings{
			ID:                  domain.GlobalSettingsID,
			RetentionPeriodDays: 90,
			UpdatedAt:           time.Now().UTC(),
			UpdatedBy:           "test-user",
		},
	}
}

// WithID sets the settings ID
func (b *SettingsBuilder) WithID(id string) *SettingsBuilder {
	b.settings.ID = id
	return b
}

// WithRetentionPeriod sets the retention period in days
func (b *SettingsBuilder) WithRetentionPeriod(days int) *SettingsBuilder {
	b.settings.RetentionPeriodDays = days
	return b
}

// WithUpdatedAt sets the updated at timestamp
func (b *SettingsBuilder) WithUpdatedAt(t time.Time) *SettingsBuilder {
	b.settings.UpdatedAt = t
	return b
}

// WithUpdatedBy sets the updated by user
func (b *SettingsBuilder) WithUpdatedBy(user string) *SettingsBuilder {
	b.settings.UpdatedBy = user
	return b
}

// Build returns the constructed settings
func (b *SettingsBuilder) Build() *domain.Settings {
	return b.settings
}

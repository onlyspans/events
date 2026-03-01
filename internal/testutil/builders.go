package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/dto"
)

type EventBuilder struct {
	event *domain.Event
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		event: &domain.Event{
			ID:         uuid.New(),
			Timestamp:  time.Now().UTC(),
			EntityID:   uuid.New(),
			EntityName: "test-entity",
			Action:     "test-action",
		},
	}
}

func (b *EventBuilder) WithID(id uuid.UUID) *EventBuilder {
	b.event.ID = id
	return b
}

func (b *EventBuilder) WithTimestamp(t time.Time) *EventBuilder {
	b.event.Timestamp = t
	return b
}

func (b *EventBuilder) WithEntityID(id uuid.UUID) *EventBuilder {
	b.event.EntityID = id
	return b
}

func (b *EventBuilder) WithEntityName(name string) *EventBuilder {
	b.event.EntityName = name
	return b
}

func (b *EventBuilder) WithAction(action string) *EventBuilder {
	b.event.Action = action
	return b
}

func (b *EventBuilder) WithUserID(userID string) *EventBuilder {
	b.event.UserID = userID
	return b
}

func (b *EventBuilder) WithIPAddress(ip string) *EventBuilder {
	b.event.IPAddress = ip
	return b
}

func (b *EventBuilder) WithUserAgent(ua string) *EventBuilder {
	b.event.UserAgent = ua
	return b
}

func (b *EventBuilder) WithTenant(tenant string) *EventBuilder {
	b.event.Tenant = tenant
	return b
}

func (b *EventBuilder) WithChanges(changes []domain.Change) *EventBuilder {
	b.event.Changes = changes
	return b
}

func (b *EventBuilder) Build() *domain.Event {
	return b.event
}

type EventDTOBuilder struct {
	dto *dto.EventDTO
}

func NewEventDTOBuilder() *EventDTOBuilder {
	return &EventDTOBuilder{
		dto: &dto.EventDTO{
			Timestamp:  time.Now().UTC(),
			EntityID:   uuid.New().String(),
			EntityName: "test-entity",
			Action:     "test-action",
		},
	}
}

func (b *EventDTOBuilder) WithTimestamp(t time.Time) *EventDTOBuilder {
	b.dto.Timestamp = t
	return b
}

func (b *EventDTOBuilder) WithEntityID(id string) *EventDTOBuilder {
	b.dto.EntityID = id
	return b
}

func (b *EventDTOBuilder) WithEntityName(name string) *EventDTOBuilder {
	b.dto.EntityName = name
	return b
}

func (b *EventDTOBuilder) WithAction(action string) *EventDTOBuilder {
	b.dto.Action = action
	return b
}

func (b *EventDTOBuilder) WithUserID(userID string) *EventDTOBuilder {
	b.dto.UserID = userID
	return b
}

func (b *EventDTOBuilder) WithIPAddress(ip string) *EventDTOBuilder {
	b.dto.IPAddress = ip
	return b
}

func (b *EventDTOBuilder) WithUserAgent(ua string) *EventDTOBuilder {
	b.dto.UserAgent = ua
	return b
}

func (b *EventDTOBuilder) WithTenant(tenant string) *EventDTOBuilder {
	b.dto.Tenant = tenant
	return b
}

func (b *EventDTOBuilder) WithChanges(changes []dto.ChangeDTO) *EventDTOBuilder {
	b.dto.Changes = changes
	return b
}

func (b *EventDTOBuilder) Build() *dto.EventDTO {
	return b.dto
}

type EventIngestRequestBuilder struct {
	request *dto.EventIngestRequest
}

func NewEventIngestRequestBuilder() *EventIngestRequestBuilder {
	return &EventIngestRequestBuilder{
		request: &dto.EventIngestRequest{
			Timestamp:  time.Now().UTC(),
			EntityID:   uuid.New().String(),
			EntityName: "test-entity",
			Action:     "test-action",
		},
	}
}

func (b *EventIngestRequestBuilder) WithTimestamp(t time.Time) *EventIngestRequestBuilder {
	b.request.Timestamp = t
	return b
}

func (b *EventIngestRequestBuilder) WithEntityID(id string) *EventIngestRequestBuilder {
	b.request.EntityID = id
	return b
}

func (b *EventIngestRequestBuilder) WithEntityName(name string) *EventIngestRequestBuilder {
	b.request.EntityName = name
	return b
}

func (b *EventIngestRequestBuilder) WithAction(action string) *EventIngestRequestBuilder {
	b.request.Action = action
	return b
}

func (b *EventIngestRequestBuilder) WithUserID(userID string) *EventIngestRequestBuilder {
	b.request.UserID = userID
	return b
}

func (b *EventIngestRequestBuilder) WithIPAddress(ip string) *EventIngestRequestBuilder {
	b.request.IPAddress = ip
	return b
}

func (b *EventIngestRequestBuilder) WithUserAgent(ua string) *EventIngestRequestBuilder {
	b.request.UserAgent = ua
	return b
}

func (b *EventIngestRequestBuilder) WithTenant(tenant string) *EventIngestRequestBuilder {
	b.request.Tenant = tenant
	return b
}

func (b *EventIngestRequestBuilder) WithChanges(changes []dto.ChangeDTO) *EventIngestRequestBuilder {
	b.request.Changes = changes
	return b
}

func (b *EventIngestRequestBuilder) Build() dto.EventIngestRequest {
	return *b.request
}

type SettingsBuilder struct {
	settings *domain.Settings
}

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

func (b *SettingsBuilder) WithID(id string) *SettingsBuilder {
	b.settings.ID = id
	return b
}

func (b *SettingsBuilder) WithRetentionPeriod(days int) *SettingsBuilder {
	b.settings.RetentionPeriodDays = days
	return b
}

func (b *SettingsBuilder) WithUpdatedAt(t time.Time) *SettingsBuilder {
	b.settings.UpdatedAt = t
	return b
}

func (b *SettingsBuilder) WithUpdatedBy(user string) *SettingsBuilder {
	b.settings.UpdatedBy = user
	return b
}

func (b *SettingsBuilder) Build() *domain.Settings {
	return b.settings
}

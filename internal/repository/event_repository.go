package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onlyspans/events/internal/domain"
	"github.com/onlyspans/events/internal/ports"
)

var _ ports.EventRepository = (*EventRepository)(nil)

type queryBuilder struct {
	conditions []string
	args       []interface{}
}

func (qb *queryBuilder) add(column string, value interface{}) {
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s = $%d", column, len(qb.args)+1))
	qb.args = append(qb.args, value)
}

func (qb *queryBuilder) addRange(column string, operator string, value interface{}) {
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s %s $%d", column, operator, len(qb.args)+1))
	qb.args = append(qb.args, value)
}

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

func (r *EventRepository) Create(ctx context.Context, event *domain.Event) (uuid.UUID, error) {
	query := `
		INSERT INTO events (
			id, timestamp, entity_id, entity_name, action, user_id,
			ip_address, user_agent, tenant, changes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		event.ID,
		event.Timestamp,
		event.EntityID,
		toNullable(event.EntityName),
		toNullable(event.Action),
		toNullable(event.UserID),
		toNullable(event.IPAddress),
		toNullable(event.UserAgent),
		toNullable(event.Tenant),
		event.Changes,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert event: %w", err)
	}

	return id, nil
}

func (r *EventRepository) SaveBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO events (
			id, timestamp, entity_id, entity_name, action, user_id,
			ip_address, user_agent, tenant, changes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	batch := &pgx.Batch{}
	for _, event := range events {
		batch.Queue(query,
			event.ID,
			event.Timestamp,
			event.EntityID,
			toNullable(event.EntityName),
			toNullable(event.Action),
			toNullable(event.UserID),
			toNullable(event.IPAddress),
			toNullable(event.UserAgent),
			toNullable(event.Tenant),
			event.Changes,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(events); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert event at index %d: %w", i, err)
		}
	}

	return nil
}

func (r *EventRepository) Search(ctx context.Context, query ports.EventSearchQuery) ([]*domain.Event, int64, error) {
	qb := &queryBuilder{}

	if query.EntityID != "" {
		entityID, err := uuid.Parse(query.EntityID)
		if err == nil {
			qb.add("entity_id", entityID)
		}
	}
	if query.EntityName != "" {
		qb.add("entity_name", query.EntityName)
	}
	if query.Action != "" {
		qb.add("action", query.Action)
	}
	if query.UserID != "" {
		qb.add("user_id", query.UserID)
	}
	if query.Tenant != "" {
		qb.add("tenant", query.Tenant)
	}
	if query.StartDate != nil {
		qb.addRange("timestamp", ">=", *query.StartDate)
	}
	if query.EndDate != nil {
		qb.addRange("timestamp", "<=", *query.EndDate)
	}

	whereClause := ""
	if len(qb.conditions) > 0 {
		whereClause = "WHERE " + strings.Join(qb.conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, qb.args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	allowedSortColumns := map[string]bool{
		"timestamp":   true,
		"entity_id":   true,
		"entity_name": true,
		"action":      true,
		"user_id":     true,
		"tenant":      true,
	}

	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "timestamp"
	}

	if !allowedSortColumns[sortBy] {
		return nil, 0, fmt.Errorf("invalid sort column: %s", sortBy)
	}

	sortOrder := strings.ToUpper(query.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	page := query.Page
	if page < 0 {
		page = 0
	}
	size := query.Size
	if size <= 0 {
		size = 20
	}
	offset := page * size

	selectQuery := fmt.Sprintf(`
		SELECT id, timestamp, entity_id, entity_name, action, user_id,
			   ip_address, user_agent, tenant, changes
		FROM events
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, len(qb.args)+1, len(qb.args)+2)

	qb.args = append(qb.args, size, offset)

	rows, err := r.pool.Query(ctx, selectQuery, qb.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var entityName, action, userID, ipAddress, userAgent, tenant *string
		var changes []domain.Change

		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.EntityID,
			&entityName,
			&action,
			&userID,
			&ipAddress,
			&userAgent,
			&tenant,
			&changes,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}

		event.EntityName = fromNullable(entityName)
		event.Action = fromNullable(action)
		event.UserID = fromNullable(userID)
		event.IPAddress = fromNullable(ipAddress)
		event.UserAgent = fromNullable(userAgent)
		event.Tenant = fromNullable(tenant)
		event.Changes = changes

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating events: %w", err)
	}

	return events, total, nil
}

func (r *EventRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM events WHERE timestamp < $1",
		cutoffDate,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", err)
	}

	return tag.RowsAffected(), nil
}

func toNullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func fromNullable(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

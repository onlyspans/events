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

// Compile-time check that EventRepository implements ports.EventRepository.
var _ ports.EventRepository = (*EventRepository)(nil)

// queryBuilder helps build SQL WHERE clauses with safe placeholder tracking.
type queryBuilder struct {
	conditions []string
	args       []interface{}
}

// add appends a condition with its argument, automatically tracking the placeholder index.
func (qb *queryBuilder) add(column string, value interface{}) {
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s = $%d", column, len(qb.args)+1))
	qb.args = append(qb.args, value)
}

// addRange appends a comparison condition (e.g., >=, <=) with its argument.
func (qb *queryBuilder) addRange(column string, operator string, value interface{}) {
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s %s $%d", column, operator, len(qb.args)+1))
	qb.args = append(qb.args, value)
}

// EventRepository handles event data access operations.
type EventRepository struct {
	pool *pgxpool.Pool
}

// NewEventRepository creates a new EventRepository.
func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

// Create inserts a single event into the database and returns its ID.
// This method is used for HTTP ingestion of individual events.
func (r *EventRepository) Create(ctx context.Context, event *domain.Event) (uuid.UUID, error) {
	query := `
		INSERT INTO events (
			id, timestamp, user_name, category, action, document_name,
			project, environment, tenant, correlation_id, trace_id, details
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query,
		event.ID,
		event.Timestamp,
		toNullable(event.User),
		toNullable(event.Category),
		toNullable(event.Action),
		toNullable(event.DocumentName),
		toNullable(event.Project),
		toNullable(event.Environment),
		toNullable(event.Tenant),
		toNullable(event.CorrelationID),
		toNullable(event.TraceID),
		event.Details,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert event: %w", err)
	}

	return id, nil
}

// SaveBatch saves multiple events using pgx Batch API for optimal performance.
// This eliminates the need for transactions and prepared statements,
// sending all inserts in a single network round trip.
func (r *EventRepository) SaveBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO events (
			id, timestamp, user_name, category, action, document_name,
			project, environment, tenant, correlation_id, trace_id, details
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	batch := &pgx.Batch{}
	for _, event := range events {
		batch.Queue(query,
			event.ID,
			event.Timestamp,
			toNullable(event.User),
			toNullable(event.Category),
			toNullable(event.Action),
			toNullable(event.DocumentName),
			toNullable(event.Project),
			toNullable(event.Environment),
			toNullable(event.Tenant),
			toNullable(event.CorrelationID),
			toNullable(event.TraceID),
			event.Details,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	// Process all results to ensure all inserts complete
	for i := 0; i < len(events); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert event at index %d: %w", i, err)
		}
	}

	return nil
}

// Search retrieves events matching the query criteria with pagination.
func (r *EventRepository) Search(ctx context.Context, query ports.EventSearchQuery) ([]*domain.Event, int64, error) {
	// Build WHERE clause using query builder
	qb := &queryBuilder{}

	if query.User != "" {
		qb.add("user_name", query.User)
	}
	if query.Category != "" {
		qb.add("category", query.Category)
	}
	if query.Action != "" {
		qb.add("action", query.Action)
	}
	if query.Document != "" {
		qb.add("document_name", query.Document)
	}
	if query.Project != "" {
		qb.add("project", query.Project)
	}
	if query.Environment != "" {
		qb.add("environment", query.Environment)
	}
	if query.Tenant != "" {
		qb.add("tenant", query.Tenant)
	}
	if query.CorrelationID != "" {
		qb.add("correlation_id", query.CorrelationID)
	}
	if query.TraceID != "" {
		qb.add("trace_id", query.TraceID)
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

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events %s", whereClause)
	var total int64
	err := r.pool.QueryRow(ctx, countQuery, qb.args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Build ORDER BY clause with whitelisted columns to prevent SQL injection
	allowedSortColumns := map[string]bool{
		"timestamp":      true,
		"user_name":      true,
		"category":       true,
		"action":         true,
		"document_name":  true,
		"project":        true,
		"environment":    true,
		"tenant":         true,
		"correlation_id": true,
		"trace_id":       true,
		"created_at":     true,
	}

	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "timestamp"
	}

	// Validate sortBy against whitelist
	if !allowedSortColumns[sortBy] {
		return nil, 0, fmt.Errorf("invalid sort column: %s", sortBy)
	}

	sortOrder := strings.ToUpper(query.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	// Add pagination
	page := query.Page
	if page < 0 {
		page = 0
	}
	size := query.Size
	if size <= 0 {
		size = 20
	}
	offset := page * size

	// Query events
	selectQuery := fmt.Sprintf(`
		SELECT id, timestamp, user_name, category, action, document_name,
			   project, environment, tenant, correlation_id, trace_id, details, created_at
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
		var userName, category, action, documentName, project, environment, tenant, correlationID, traceID *string
		var details *domain.EventDetails

		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&userName,
			&category,
			&action,
			&documentName,
			&project,
			&environment,
			&tenant,
			&correlationID,
			&traceID,
			&details,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}

		event.User = fromNullable(userName)
		event.Category = fromNullable(category)
		event.Action = fromNullable(action)
		event.DocumentName = fromNullable(documentName)
		event.Project = fromNullable(project)
		event.Environment = fromNullable(environment)
		event.Tenant = fromNullable(tenant)
		event.CorrelationID = fromNullable(correlationID)
		event.TraceID = fromNullable(traceID)
		event.Details = details

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating events: %w", err)
	}

	return events, total, nil
}

// DeleteOlderThan deletes events older than the specified cutoff date.
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

// toNullable converts an empty string to nil for pgx nullable handling.
func toNullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// fromNullable converts a nullable string pointer to a string value.
func fromNullable(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

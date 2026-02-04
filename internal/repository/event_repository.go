package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onlyspans/events/internal/domain"
)

// EventRepository handles event data access operations.
type EventRepository struct {
	db *sql.DB
}

// NewEventRepository creates a new EventRepository.
func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
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
	err := r.db.QueryRowContext(ctx, query,
		event.ID,
		event.Timestamp,
		nullString(event.User),
		nullString(event.Category),
		nullString(event.Action),
		nullString(event.DocumentName),
		nullString(event.Project),
		nullString(event.Environment),
		nullString(event.Tenant),
		nullString(event.CorrelationID),
		nullString(event.TraceID),
		event.Details,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert event: %w", err)
	}

	return id, nil
}

// SaveBatch saves multiple events in a single transaction for efficiency.
func (r *EventRepository) SaveBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (
			id, timestamp, user_name, category, action, document_name,
			project, environment, tenant, correlation_id, trace_id, details
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		_, err := stmt.ExecContext(ctx,
			event.ID,
			event.Timestamp,
			nullString(event.User),
			nullString(event.Category),
			nullString(event.Action),
			nullString(event.DocumentName),
			nullString(event.Project),
			nullString(event.Environment),
			nullString(event.Tenant),
			nullString(event.CorrelationID),
			nullString(event.TraceID),
			event.Details,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SearchQuery represents search criteria for events.
type SearchQuery struct {
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

// Search retrieves events matching the query criteria with pagination.
func (r *EventRepository) Search(ctx context.Context, query SearchQuery) ([]*domain.Event, int64, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if query.User != "" {
		conditions = append(conditions, fmt.Sprintf("user_name = $%d", argIndex))
		args = append(args, query.User)
		argIndex++
	}
	if query.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, query.Category)
		argIndex++
	}
	if query.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, query.Action)
		argIndex++
	}
	if query.Document != "" {
		conditions = append(conditions, fmt.Sprintf("document_name = $%d", argIndex))
		args = append(args, query.Document)
		argIndex++
	}
	if query.Project != "" {
		conditions = append(conditions, fmt.Sprintf("project = $%d", argIndex))
		args = append(args, query.Project)
		argIndex++
	}
	if query.Environment != "" {
		conditions = append(conditions, fmt.Sprintf("environment = $%d", argIndex))
		args = append(args, query.Environment)
		argIndex++
	}
	if query.Tenant != "" {
		conditions = append(conditions, fmt.Sprintf("tenant = $%d", argIndex))
		args = append(args, query.Tenant)
		argIndex++
	}
	if query.CorrelationID != "" {
		conditions = append(conditions, fmt.Sprintf("correlation_id = $%d", argIndex))
		args = append(args, query.CorrelationID)
		argIndex++
	}
	if query.TraceID != "" {
		conditions = append(conditions, fmt.Sprintf("trace_id = $%d", argIndex))
		args = append(args, query.TraceID)
		argIndex++
	}
	if query.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *query.StartDate)
		argIndex++
	}
	if query.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *query.EndDate)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
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
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, size, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		var userName, category, action, documentName, project, environment, tenant, correlationID, traceID sql.NullString
		var details sql.NullString

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

		event.User = userName.String
		event.Category = category.String
		event.Action = action.String
		event.DocumentName = documentName.String
		event.Project = project.String
		event.Environment = environment.String
		event.Tenant = tenant.String
		event.CorrelationID = correlationID.String
		event.TraceID = traceID.String

		if details.Valid && details.String != "" {
			var eventDetails domain.EventDetails
			if err := eventDetails.Scan([]byte(details.String)); err == nil {
				event.Details = &eventDetails
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating events: %w", err)
	}

	return events, total, nil
}

// DeleteOlderThan deletes events older than the specified cutoff date.
func (r *EventRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM events WHERE timestamp < $1",
		cutoffDate,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// nullString converts an empty string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

package handler

import (
	"context"
	"database/sql"

	"github.com/onlyspans/events/internal/ports"
)

// DBHealthChecker adapts *sql.DB to the ports.HealthChecker interface.
type DBHealthChecker struct {
	db *sql.DB
}

// Compile-time interface verification.
var _ ports.HealthChecker = (*DBHealthChecker)(nil)

// NewDBHealthChecker creates a new DBHealthChecker.
func NewDBHealthChecker(db *sql.DB) *DBHealthChecker {
	return &DBHealthChecker{db: db}
}

// Ping checks database connectivity.
func (c *DBHealthChecker) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

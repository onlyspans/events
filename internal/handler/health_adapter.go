package handler

import (
	"context"
	"database/sql"

	"github.com/onlyspans/events/internal/ports"
)

type DBHealthChecker struct {
	db *sql.DB
}

var _ ports.HealthChecker = (*DBHealthChecker)(nil)

func NewDBHealthChecker(db *sql.DB) *DBHealthChecker {
	return &DBHealthChecker{db: db}
}

func (c *DBHealthChecker) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

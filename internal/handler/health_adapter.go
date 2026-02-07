package handler

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onlyspans/events/internal/ports"
)

type DBHealthChecker struct {
	pool *pgxpool.Pool
}

var _ ports.HealthChecker = (*DBHealthChecker)(nil)

func NewDBHealthChecker(pool *pgxpool.Pool) *DBHealthChecker {
	return &DBHealthChecker{pool: pool}
}

func (c *DBHealthChecker) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

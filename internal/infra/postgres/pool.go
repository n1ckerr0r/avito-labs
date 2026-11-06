// Package postgres contains the PostgreSQL connection pool and executor helper.
package postgres

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/n1ckerr0r/avito-labs/internal/config"
)

type txKey struct{}

// OpenPool creates a PostgreSQL connection pool and verifies it with Ping.
// tp and mp instrument every query with otelpgx: a span and a
// db.client.operation.duration measurement per call, without the repository
// knowing telemetry exists.
func OpenPool(ctx context.Context, cfg config.Config, tp trace.TracerProvider, mp metric.MeterProvider) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns
	poolConfig.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
	poolConfig.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(tp),
		otelpgx.WithMeterProvider(mp),
		otelpgx.WithMetricOperationName(otelpgx.SQLOperationName),
	)

	connectCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}

// Executor returns the transaction attached to ctx, or the pool if there is none.
func Executor(ctx context.Context, pool *pgxpool.Pool) interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
} {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return pool
}

// WithTx attaches tx to ctx so Executor returns it.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext reports whether ctx already carries a transaction.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(context.Context, func(context.Context) error) error
}

type txKey struct{}

// Manager uses Read Committed: PostgreSQL's default is sufficient because the
// database unique index and conditional UPDATE provide the concurrency guards.
type Manager struct{ pool *pgxpool.Pool }

func NewTxManager(pool *pgxpool.Pool) *Manager { return &Manager{pool: pool} }

func (m *Manager) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback(ctx)
			panic(recovered)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()
	return fn(context.WithValue(ctx, txKey{}, tx))
}

// Executor returns the transaction attached by Manager, or the pool for reads
// and calls made outside a transaction.
func Executor(ctx context.Context, pool *pgxpool.Pool) interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
} {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

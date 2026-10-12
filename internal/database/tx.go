package database

import (
	"context"
	"fmt"
	"time"

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
type Manager struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, timeout time.Duration) *Manager {
	return &Manager{pool: pool, timeout: timeout}
}

func (m *Manager) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	if ctx.Value(txKey{}) != nil {
		return fn(ctx)
	}

	txCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	tx, err := m.pool.BeginTx(txCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			m.rollback(tx)

			panic(recovered)
		}

		if err != nil {
			m.rollback(tx)
			return
		}

		if err = tx.Commit(txCtx); err != nil {
			m.rollback(tx)

			err = fmt.Errorf("commit transaction: %w", err)
		}
	}()

	return fn(context.WithValue(txCtx, txKey{}, tx))
}

// rollback runs independently of the request context, which may already be
// cancelled. Its own timeout prevents cleanup from blocking shutdown forever.
func (m *Manager) rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	_ = tx.Rollback(ctx)
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

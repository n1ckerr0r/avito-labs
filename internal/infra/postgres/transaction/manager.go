// Package transaction provides a transaction manager behind a narrow interface.
package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres"
)

// Manager uses Read Committed; callers needing more take explicit row locks.
type Manager struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func New(pool *pgxpool.Pool, timeout time.Duration) *Manager {
	return &Manager{pool: pool, timeout: timeout}
}

func (m *Manager) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	if _, ok := postgres.TxFromContext(ctx); ok {
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

	return fn(postgres.WithTx(txCtx, tx))
}

// rollback uses its own timeout since ctx may already be cancelled.
func (m *Manager) rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	_ = tx.Rollback(ctx)
}

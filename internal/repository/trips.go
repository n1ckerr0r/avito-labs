package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1ckerr0r/avito-labs/internal/database"
	"github.com/n1ckerr0r/avito-labs/internal/domain"
)

var sql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type Trips struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewTrips(pool *pgxpool.Pool, timeout time.Duration) *Trips {
	return &Trips{pool: pool, timeout: timeout}
}

func (r *Trips) Create(ctx context.Context, trip domain.Trip) error {
	q, args, err := sql.Insert("trips").Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(trip.ID, trip.UserID, trip.DriverID, trip.StartPoint.Latitude, trip.StartPoint.Longitude, trip.EndPoint.Latitude, trip.EndPoint.Longitude, trip.Price, trip.Status, trip.StartedAt).ToSql()
	if err != nil {
		return fmt.Errorf("build create trip: %w", err)
	}
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()
	_, err = database.Executor(ctx, r.pool).Exec(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Trips) AddStatusHistory(ctx context.Context, id uuid.UUID, from *string, to, reason string, changedAt time.Time) error {
	q, args, err := sql.Insert("trip_status_history").Columns("trip_id", "from_status", "to_status", "reason", "changed_at").Values(id, from, to, reason, changedAt).ToSql()
	if err != nil {
		return fmt.Errorf("build status history: %w", err)
	}
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()
	_, err = database.Executor(ctx, r.pool).Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}
	return nil
}

// ClaimIdempotencyKey reserves a client key in the current transaction. The
// unique primary key makes concurrent requests with the same key converge.
func (r *Trips) ClaimIdempotencyKey(ctx context.Context, key uuid.UUID, hash []byte) (claimed bool, existingHash []byte, existingTripID uuid.UUID, err error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()
	// Idempotency records are retained for 24 hours. This opportunistic cleanup
	// keeps the table bounded without a separate worker in this small service.
	cleanup, cleanupArgs, buildErr := sql.Delete("idempotency_keys").Where("created_at < now() - interval '24 hours'").ToSql()
	if buildErr != nil {
		return false, nil, uuid.Nil, fmt.Errorf("build idempotency cleanup: %w", buildErr)
	}
	exec := database.Executor(ctx, r.pool)
	if _, err = exec.Exec(ctx, cleanup, cleanupArgs...); err != nil {
		return false, nil, uuid.Nil, fmt.Errorf("delete expired idempotency keys: %w", err)
	}
	q, args, buildErr := sql.Insert("idempotency_keys").Columns("key", "request_hash", "trip_id").Values(key, hash, nil).Suffix("ON CONFLICT (key) DO NOTHING RETURNING key").ToSql()
	if buildErr != nil {
		return false, nil, uuid.Nil, fmt.Errorf("build idempotency claim: %w", buildErr)
	}
	var returned uuid.UUID
	err = exec.QueryRow(ctx, q, args...).Scan(&returned)
	if err == nil {
		return true, nil, uuid.Nil, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, nil, uuid.Nil, fmt.Errorf("claim idempotency key: %w", err)
	}
	q, args, buildErr = sql.Select("request_hash", "trip_id").From("idempotency_keys").Where(squirrel.Eq{"key": key}).ToSql()
	if buildErr != nil {
		return false, nil, uuid.Nil, fmt.Errorf("build idempotency lookup: %w", buildErr)
	}
	if err = exec.QueryRow(ctx, q, args...).Scan(&existingHash, &existingTripID); err != nil {
		return false, nil, uuid.Nil, fmt.Errorf("lookup idempotency key: %w", err)
	}
	return false, existingHash, existingTripID, nil
}

func (r *Trips) CompleteIdempotencyKey(ctx context.Context, key, tripID uuid.UUID) error {
	q, args, err := sql.Update("idempotency_keys").Set("trip_id", tripID).Where(squirrel.Eq{"key": key}).ToSql()
	if err != nil {
		return fmt.Errorf("build idempotency completion: %w", err)
	}
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()
	if _, err = database.Executor(ctx, r.pool).Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	return nil
}

func (r *Trips) Get(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	q, args, err := sql.Select("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").From("trips").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build get trip: %w", err)
	}
	return r.scan(ctx, q, args...)
}

func (r *Trips) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domain.Trip, error) {
	q, args, err := sql.Update("trips").Set("status", "completed").Set("finished_at", finishedAt).Set("updated_at", finishedAt).
		Where(squirrel.Eq{"id": id, "status": "active"}).Suffix("RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at").ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build finish trip: %w", err)
	}
	trip, err := r.scan(ctx, q, args...)
	if !errors.Is(err, domain.ErrNotFound) {
		return trip, err
	}
	// The conditional update closes the race. A follow-up read only classifies
	// its zero-row result for the client.
	_, getErr := r.Get(ctx, id)
	if errors.Is(getErr, domain.ErrNotFound) {
		return domain.Trip{}, domain.ErrNotFound
	}
	if getErr != nil {
		return domain.Trip{}, getErr
	}
	return domain.Trip{}, domain.ErrCompleted
}

func (r *Trips) scan(ctx context.Context, query string, args ...any) (domain.Trip, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()
	var t domain.Trip
	err := database.Executor(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&t.ID, &t.UserID, &t.DriverID, &t.StartPoint.Latitude, &t.StartPoint.Longitude, &t.EndPoint.Latitude, &t.EndPoint.Longitude, &t.Price, &t.Status, &t.StartedAt, &t.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Trip{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Trip{}, fmt.Errorf("query trip: %w", err)
	}
	return t, nil
}

func (r *Trips) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.timeout)
}

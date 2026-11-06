package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	domaintrip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

// Create inserts a new trip; a unique-violation maps to ErrDriverBusy or ErrConflict.
func (r *Repository) Create(ctx context.Context, t model.Trip) error {
	q, args, err := sql.Insert("trips").Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.StartPoint.Latitude, t.StartPoint.Longitude, t.EndPoint.Latitude, t.EndPoint.Longitude, t.Price, t.Status, t.StartedAt).ToSql()
	if err != nil {
		return fmt.Errorf("build create trip: %w", err)
	}

	_, err = postgres.Executor(ctx, r.pool).Exec(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "trips_active_driver_idx" {
				return domaintrip.ErrDriverBusy
			}

			return domaintrip.ErrConflict
		}

		return fmt.Errorf("insert trip: %w", err)
	}

	return nil
}

// Update persists a trip previously loaded with GetForUpdate.
func (r *Repository) Update(ctx context.Context, t model.Trip) error {
	q, args, err := sql.Update("trips").
		Set("status", t.Status).
		Set("finished_at", t.FinishedAt).
		Set("updated_at", t.FinishedAt).
		Where("id = ?", t.ID).ToSql()
	if err != nil {
		return fmt.Errorf("build update trip: %w", err)
	}

	if _, err = postgres.Executor(ctx, r.pool).Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("update trip: %w", err)
	}

	return nil
}

//go:build integration

package trip_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domaintrip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres/transaction"
	"github.com/n1ckerr0r/avito-labs/internal/model"
	"github.com/n1ckerr0r/avito-labs/internal/repository/trip"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func newTrip(driverID uuid.UUID) model.Trip {
	return model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: model.Coordinates{Latitude: 1, Longitude: 1},
		EndPoint:   model.Coordinates{Latitude: 2, Longitude: 2},
		Price:      500,
		Status:     string(domaintrip.StatusActive),
		StartedAt:  time.Now().UTC(),
	}
}

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id)
	})
}

func TestRepository_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	repo := trip.New(pool)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	cleanupTrip(t, pool, tr.ID)

	require.NoError(t, repo.Create(ctx, tr))

	got, err := repo.Get(ctx, tr.ID)
	require.NoError(t, err)
	assert.Equal(t, tr.ID, got.ID)
	assert.Equal(t, tr.DriverID, got.DriverID)
	assert.Equal(t, string(domaintrip.StatusActive), got.Status)
}

func TestRepository_Get_NotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := trip.New(pool)

	_, err := repo.Get(context.Background(), uuid.New())

	require.ErrorIs(t, err, domaintrip.ErrTripNotFound)
}

func TestRepository_Create_DriverBusy(t *testing.T) {
	pool := newTestPool(t)
	repo := trip.New(pool)
	ctx := context.Background()

	driverID := uuid.New()
	first := newTrip(driverID)
	cleanupTrip(t, pool, first.ID)
	require.NoError(t, repo.Create(ctx, first))

	second := newTrip(driverID)
	cleanupTrip(t, pool, second.ID)
	err := repo.Create(ctx, second)

	require.ErrorIs(t, err, domaintrip.ErrDriverBusy)
}

func TestRepository_GetForUpdateAndUpdate(t *testing.T) {
	pool := newTestPool(t)
	repo := trip.New(pool)
	txManager := transaction.New(pool, 5*time.Second)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	cleanupTrip(t, pool, tr.ID)
	require.NoError(t, repo.Create(ctx, tr))

	finishedAt := time.Now().UTC()

	err := txManager.Do(ctx, func(ctx context.Context) error {
		locked, err := repo.GetForUpdate(ctx, tr.ID)
		if err != nil {
			return err
		}

		locked.Status = string(domaintrip.StatusCompleted)
		locked.FinishedAt = &finishedAt

		return repo.Update(ctx, locked)
	})
	require.NoError(t, err)

	got, err := repo.Get(ctx, tr.ID)
	require.NoError(t, err)
	assert.Equal(t, string(domaintrip.StatusCompleted), got.Status)
	require.NotNil(t, got.FinishedAt)
}

func TestRepository_IdempotencyKey_ClaimAndComplete(t *testing.T) {
	pool := newTestPool(t)
	repo := trip.New(pool)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	cleanupTrip(t, pool, tr.ID)
	require.NoError(t, repo.Create(ctx, tr))

	key := uuid.New()
	hash := []byte("request-hash")

	claimed, _, _, err := repo.ClaimIdempotencyKey(ctx, key, hash)
	require.NoError(t, err)
	assert.True(t, claimed)

	require.NoError(t, repo.CompleteIdempotencyKey(ctx, key, tr.ID))

	claimed, existingHash, existingTripID, err := repo.ClaimIdempotencyKey(ctx, key, hash)
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Equal(t, hash, existingHash)
	assert.Equal(t, tr.ID, existingTripID)
}

// TestRepository_Create_RollsBackOnError verifies that a failing second
// insert in the same transaction leaves no trace of the first.
func TestRepository_Create_RollsBackOnError(t *testing.T) {
	pool := newTestPool(t)
	repo := trip.New(pool)
	txManager := transaction.New(pool, 5*time.Second)
	ctx := context.Background()

	tr := newTrip(uuid.New())
	cleanupTrip(t, pool, tr.ID)

	err := txManager.Do(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, tr); err != nil {
			return err
		}

		// Violates the to_status CHECK constraint, forcing a rollback.
		return repo.AddStatusHistory(ctx, tr.ID, nil, "bogus-status", "test", tr.StartedAt)
	})
	require.Error(t, err)

	_, getErr := repo.Get(ctx, tr.ID)
	require.ErrorIs(t, getErr, domaintrip.ErrTripNotFound)
}

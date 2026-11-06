package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
	"github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
)

// passthroughTx runs fn with the given ctx, same as the real transaction.Manager would.
func passthroughTx(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).AnyTimes()
}

func baseInput() create.Input {
	return create.Input{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: trip.Coordinates{Latitude: 55.7, Longitude: 37.6},
		EndPoint:   trip.Coordinates{Latitude: 55.8, Longitude: 37.7},
		Price:      500,
	}
}

func TestUsecase_Execute_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().AddStatusHistory(gomock.Any(), gomock.Any(), nil, string(trip.StatusActive), gomock.Any(), gomock.Any()).Return(nil)

	uc := create.New(repo, tx)
	result, created, err := uc.Execute(context.Background(), baseInput())

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, string(trip.StatusActive), result.Status)
}

func TestUsecase_Execute_DriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(trip.ErrDriverBusy)

	uc := create.New(repo, tx)
	_, created, err := uc.Execute(context.Background(), baseInput())

	require.ErrorIs(t, err, trip.ErrDriverBusy)
	assert.False(t, created)
}

func TestUsecase_Execute_IdempotentReplay(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	key := uuid.New()
	existingID := uuid.New()
	hash := []byte("same-hash")
	existing := model.Trip{ID: existingID, Status: string(trip.StatusActive)}

	repo.EXPECT().ClaimIdempotencyKey(gomock.Any(), key, hash).Return(false, hash, existingID, nil)
	repo.EXPECT().Get(gomock.Any(), existingID).Return(existing, nil)

	in := baseInput()
	in.IdempotencyKey = &key
	in.RequestHash = hash

	uc := create.New(repo, tx)
	result, created, err := uc.Execute(context.Background(), in)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, existingID, result.ID)
}

func TestUsecase_Execute_IdempotencyConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	key := uuid.New()

	repo.EXPECT().ClaimIdempotencyKey(gomock.Any(), key, []byte("hash-a")).Return(false, []byte("hash-b"), uuid.New(), nil)

	in := baseInput()
	in.IdempotencyKey = &key
	in.RequestHash = []byte("hash-a")

	uc := create.New(repo, tx)
	_, created, err := uc.Execute(context.Background(), in)

	require.ErrorIs(t, err, trip.ErrIdempotencyConflict)
	assert.False(t, created)
}

func TestUsecase_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	boom := errors.New("connection reset")
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(boom)

	uc := create.New(repo, tx)
	_, _, err := uc.Execute(context.Background(), baseInput())

	require.ErrorIs(t, err, boom)
}

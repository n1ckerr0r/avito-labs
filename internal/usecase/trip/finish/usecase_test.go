package finish_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
	"github.com/n1ckerr0r/avito-labs/internal/usecase/trip/finish"
)

func passthroughTx(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).AnyTimes()
}

func TestUsecase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	id := uuid.New()
	active := model.Trip{ID: id, Status: string(trip.StatusActive), StartedAt: time.Now().UTC()}

	repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(active, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, m model.Trip) error {
		assert.Equal(t, string(trip.StatusCompleted), m.Status)
		assert.NotNil(t, m.FinishedAt)

		return nil
	})
	repo.EXPECT().AddStatusHistory(gomock.Any(), id, gomock.Any(), string(trip.StatusCompleted), gomock.Any(), gomock.Any()).Return(nil)

	uc := finish.New(repo, tx)
	result, err := uc.Execute(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, string(trip.StatusCompleted), result.Status)
}

func TestUsecase_Execute_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	id := uuid.New()
	completed := model.Trip{ID: id, Status: string(trip.StatusCompleted)}

	repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(completed, nil)

	uc := finish.New(repo, tx)
	_, err := uc.Execute(context.Background(), id)

	require.ErrorIs(t, err, trip.ErrTripCompleted)
}

func TestUsecase_Execute_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	passthroughTx(tx)

	id := uuid.New()
	repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)

	uc := finish.New(repo, tx)
	_, err := uc.Execute(context.Background(), id)

	require.ErrorIs(t, err, trip.ErrTripNotFound)
}

package get_test

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
	"github.com/n1ckerr0r/avito-labs/internal/usecase/trip/get"
)

func TestUsecase_Execute_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	id := uuid.New()
	want := model.Trip{ID: id, Status: string(trip.StatusActive)}

	repo.EXPECT().Get(gomock.Any(), id).Return(want, nil)

	uc := get.New(repo, time.Second)
	got, err := uc.Execute(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUsecase_Execute_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	id := uuid.New()

	repo.EXPECT().Get(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)

	uc := get.New(repo, time.Second)
	_, err := uc.Execute(context.Background(), id)

	require.ErrorIs(t, err, trip.ErrTripNotFound)
}

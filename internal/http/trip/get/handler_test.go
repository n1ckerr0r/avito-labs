package get_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/trip/get"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

func TestHandler_Handle_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripGetter(ctrl)

	id := uuid.New()
	uc.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{ID: id, Status: string(trip.StatusActive)}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	rec := httptest.NewRecorder()

	get.New(uc).Handle(rec, req, id)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Handle_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripGetter(ctrl)

	id := uuid.New()
	uc.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	rec := httptest.NewRecorder()

	get.New(uc).Handle(rec, req, id)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var problem api.Problem

	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "trip_not_found", problem.Code)
}

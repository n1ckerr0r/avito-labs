package finish_test

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
	"github.com/n1ckerr0r/avito-labs/internal/http/trip/finish"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

func newHandler(ctrl *gomock.Controller, uc *MocktripFinisher) *finish.Handler {
	metrics := NewMockmetrics(ctrl)
	metrics.EXPECT().IncCompleted(gomock.Any()).AnyTimes()

	logger := NewMocklogger(ctrl)
	logger.EXPECT().InfoContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	return finish.New(uc, metrics, logger)
}

func TestHandler_Handle_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripFinisher(ctrl)

	id := uuid.New()
	uc.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{ID: id, Status: string(trip.StatusCompleted)}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	newHandler(ctrl, uc).Handle(rec, req, id)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Handle_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripFinisher(ctrl)

	id := uuid.New()
	uc.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripCompleted)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	rec := httptest.NewRecorder()

	newHandler(ctrl, uc).Handle(rec, req, id)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var problem api.Problem

	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "trip_completed", problem.Code)
}

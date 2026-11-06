package create_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/trip/create"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

const validBody = `{"user_id":"11111111-1111-1111-1111-111111111111","driver_id":"22222222-2222-2222-2222-222222222222","start_point":{"latitude":1,"longitude":1},"end_point":{"latitude":2,"longitude":2},"price":500}`

func newHandler(ctrl *gomock.Controller, uc *MocktripCreator) *create.Handler {
	metrics := NewMockmetrics(ctrl)
	metrics.EXPECT().IncCreated(gomock.Any()).AnyTimes()

	logger := NewMocklogger(ctrl)
	logger.EXPECT().InfoContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	return create.New(uc, metrics, logger)
}

func doRequest(t *testing.T, h *create.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Handle(rec, req, api.CreateTripParams{})

	return rec
}

func TestHandler_Handle_Created(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripCreator(ctrl)

	tripID := uuid.New()
	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{ID: tripID, Status: string(trip.StatusActive)}, true, nil)

	rec := doRequest(t, newHandler(ctrl, uc), validBody)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "/api/v1/trips/"+tripID.String(), rec.Header().Get("Location"))
}

func TestHandler_Handle_IdempotentReplay(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripCreator(ctrl)

	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{ID: uuid.New(), Status: string(trip.StatusActive)}, false, nil)

	rec := doRequest(t, newHandler(ctrl, uc), validBody)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Handle_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripCreator(ctrl)
	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(0)

	rec := doRequest(t, newHandler(ctrl, uc), `{}`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var problem api.Problem

	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "invalid_request", problem.Code)
}

func TestHandler_Handle_DomainError(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripCreator(ctrl)
	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{}, false, trip.ErrDriverBusy)

	rec := doRequest(t, newHandler(ctrl, uc), validBody)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var problem api.Problem

	require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
	assert.Equal(t, "driver_busy", problem.Code)
}

func TestHandler_Handle_RepositoryErrorHidesDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMocktripCreator(ctrl)
	uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{}, false, assert.AnError)

	rec := doRequest(t, newHandler(ctrl, uc), validBody)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), assert.AnError.Error())
}

package trip_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
)

func validCoordinates() trip.Coordinates {
	return trip.Coordinates{Latitude: 55.7, Longitude: 37.6}
}

func TestNewTrip(t *testing.T) {
	userID, driverID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name      string
		userID    uuid.UUID
		driverID  uuid.UUID
		start     trip.Coordinates
		end       trip.Coordinates
		price     int64
		wantField string
	}{
		{name: "valid", userID: userID, driverID: driverID, start: validCoordinates(), end: validCoordinates(), price: 500},
		{name: "zero price is valid", userID: userID, driverID: driverID, start: validCoordinates(), end: validCoordinates(), price: 0},
		{name: "latitude -90 is valid", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: -90, Longitude: 0}, end: validCoordinates(), price: 500},
		{name: "latitude 90 is valid", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: 90, Longitude: 0}, end: validCoordinates(), price: 500},
		{name: "longitude -180 is valid", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: 0, Longitude: -180}, end: validCoordinates(), price: 500},
		{name: "longitude 180 is valid", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: 0, Longitude: 180}, end: validCoordinates(), price: 500},
		{name: "nil user id", userID: uuid.Nil, driverID: driverID, start: validCoordinates(), end: validCoordinates(), price: 500, wantField: "user_id"},
		{name: "nil driver id", userID: userID, driverID: uuid.Nil, start: validCoordinates(), end: validCoordinates(), price: 500, wantField: "driver_id"},
		{name: "start latitude too high", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: 91, Longitude: 0}, end: validCoordinates(), price: 500, wantField: "start_point.latitude"},
		{name: "start latitude too low", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: -91, Longitude: 0}, end: validCoordinates(), price: 500, wantField: "start_point.latitude"},
		{name: "start longitude out of range", userID: userID, driverID: driverID, start: trip.Coordinates{Latitude: 0, Longitude: 181}, end: validCoordinates(), price: 500, wantField: "start_point.longitude"},
		{name: "end latitude out of range", userID: userID, driverID: driverID, start: validCoordinates(), end: trip.Coordinates{Latitude: -91, Longitude: 0}, price: 500, wantField: "end_point.latitude"},
		{name: "end longitude out of range", userID: userID, driverID: driverID, start: validCoordinates(), end: trip.Coordinates{Latitude: 0, Longitude: -181}, price: 500, wantField: "end_point.longitude"},
		{name: "negative price", userID: userID, driverID: driverID, start: validCoordinates(), end: validCoordinates(), price: -1, wantField: "price"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := trip.NewTrip(uuid.New(), tt.userID, tt.driverID, tt.start, tt.end, tt.price, now)

			if tt.wantField == "" {
				require.NoError(t, err)
				assert.Equal(t, trip.StatusActive, got.Status)
				assert.Equal(t, tt.userID, got.UserID)
				assert.Equal(t, tt.driverID, got.DriverID)
				assert.Nil(t, got.FinishedAt)

				return
			}

			var validationErr *trip.ValidationError

			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.wantField, validationErr.Field)
		})
	}
}

func TestTrip_Finish(t *testing.T) {
	t.Run("active trip finishes", func(t *testing.T) {
		tr, err := trip.NewTrip(uuid.New(), uuid.New(), uuid.New(), validCoordinates(), validCoordinates(), 500, time.Now().UTC())
		require.NoError(t, err)

		at := time.Now().UTC()
		err = tr.Finish(at)

		require.NoError(t, err)
		assert.Equal(t, trip.StatusCompleted, tr.Status)
		require.NotNil(t, tr.FinishedAt)
		assert.True(t, tr.FinishedAt.Equal(at))
	})

	t.Run("already completed trip cannot finish again", func(t *testing.T) {
		tr, err := trip.NewTrip(uuid.New(), uuid.New(), uuid.New(), validCoordinates(), validCoordinates(), 500, time.Now().UTC())
		require.NoError(t, err)
		require.NoError(t, tr.Finish(time.Now().UTC()))

		err = tr.Finish(time.Now().UTC())

		require.ErrorIs(t, err, trip.ErrTripCompleted)
	})
}

package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Coordinates struct{ Latitude, Longitude float64 }

type Trip struct {
	ID, UserID, DriverID uuid.UUID
	StartPoint, EndPoint Coordinates
	Price                int64
	Status               Status
	StartedAt            time.Time
	FinishedAt           *time.Time
}

// NewTrip builds a Trip and enforces its business invariants.
func NewTrip(id, userID, driverID uuid.UUID, start, end Coordinates, price int64, startedAt time.Time) (Trip, error) {
	if userID == uuid.Nil {
		return Trip{}, &ValidationError{Field: "user_id", Reason: "must be a non-empty UUID"}
	}

	if driverID == uuid.Nil {
		return Trip{}, &ValidationError{Field: "driver_id", Reason: "must be a non-empty UUID"}
	}

	if err := validateCoordinates("start_point", start); err != nil {
		return Trip{}, err
	}

	if err := validateCoordinates("end_point", end); err != nil {
		return Trip{}, err
	}

	if price < 0 {
		return Trip{}, &ValidationError{Field: "price", Reason: "must not be negative"}
	}

	return Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: start,
		EndPoint:   end,
		Price:      price,
		Status:     StatusActive,
		StartedAt:  startedAt,
	}, nil
}

func validateCoordinates(field string, c Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return &ValidationError{Field: field + ".latitude", Reason: "must be between -90 and 90"}
	}

	if c.Longitude < -180 || c.Longitude > 180 {
		return &ValidationError{Field: field + ".longitude", Reason: "must be between -180 and 180"}
	}

	return nil
}

// Finish transitions an active trip to completed.
func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}

	t.Status = StatusCompleted
	t.FinishedAt = &at

	return nil
}

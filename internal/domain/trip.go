package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound            = errors.New("trip not found")
	ErrCompleted           = errors.New("trip already completed")
	ErrDriverBusy          = errors.New("driver already has an active trip")
	ErrConflict            = errors.New("trip conflict")
	ErrIdempotencyConflict = errors.New("idempotency key was used with a different request")
)

type Coordinates struct{ Latitude, Longitude float64 }

type Trip struct {
	ID, UserID, DriverID uuid.UUID
	StartPoint, EndPoint Coordinates
	Price                int64
	Status               string
	StartedAt            time.Time
	FinishedAt           *time.Time
}

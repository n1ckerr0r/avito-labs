// Package trip contains the trip domain model: state, rules and errors.
package trip

import "errors"

var (
	ErrTripNotFound        = errors.New("trip not found")
	ErrTripCompleted       = errors.New("trip already completed")
	ErrDriverBusy          = errors.New("driver already has an active trip")
	ErrConflict            = errors.New("trip conflict")
	ErrIdempotencyConflict = errors.New("idempotency key was used with a different request")
)

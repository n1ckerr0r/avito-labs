package domainerror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", trip.ErrTripNotFound, 404, "trip_not_found"},
		{"wrapped not found", fmt.Errorf("lookup trip: %w", trip.ErrTripNotFound), 404, "trip_not_found"},
		{"completed", trip.ErrTripCompleted, 409, "trip_completed"},
		{"driver busy", trip.ErrDriverBusy, 409, "driver_busy"},
		{"conflict", trip.ErrConflict, 409, "conflict"},
		{"idempotency conflict", trip.ErrIdempotencyConflict, 409, "idempotency_conflict"},
		{"validation error", &trip.ValidationError{Field: "price", Reason: "must not be negative"}, 400, "invalid_request"},
		{"unknown error has no leaked details", errors.New("pq: connection refused"), 500, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, problem := domainerror.Problem(tt.err)

			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantCode, problem.Code)

			if tt.wantCode == "internal_error" {
				assert.NotContains(t, *problem.Detail, "pq:")
			}
		})
	}
}

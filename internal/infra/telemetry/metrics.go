package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// TripMetrics tracks the business counters not already visible in the HTTP metrics.
type TripMetrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewTripMetrics(meter metric.Meter) (*TripMetrics, error) {
	created, err := meter.Int64Counter("tripgo.trips.created", metric.WithDescription("Trips created"), metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create tripgo.trips.created counter: %w", err)
	}

	completed, err := meter.Int64Counter("tripgo.trips.completed", metric.WithDescription("Trips completed"), metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create tripgo.trips.completed counter: %w", err)
	}

	return &TripMetrics{created: created, completed: completed}, nil
}

func (m *TripMetrics) IncCreated(ctx context.Context) {
	m.created.Add(ctx, 1)
}

func (m *TripMetrics) IncCompleted(ctx context.Context) {
	m.completed.Add(ctx, 1)
}

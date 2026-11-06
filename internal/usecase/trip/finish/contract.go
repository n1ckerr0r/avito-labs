//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type (
	tripRepository interface {
		GetForUpdate(ctx context.Context, id uuid.UUID) (model.Trip, error)
		Update(ctx context.Context, trip model.Trip) error
		AddStatusHistory(ctx context.Context, id uuid.UUID, from *string, to, reason string, changedAt time.Time) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)

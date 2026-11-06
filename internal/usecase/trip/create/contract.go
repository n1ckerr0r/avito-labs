//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type (
	tripRepository interface {
		ClaimIdempotencyKey(ctx context.Context, key uuid.UUID, hash []byte) (claimed bool, existingHash []byte, existingTripID uuid.UUID, err error)
		CompleteIdempotencyKey(ctx context.Context, key, tripID uuid.UUID) error
		Create(ctx context.Context, trip model.Trip) error
		AddStatusHistory(ctx context.Context, id uuid.UUID, from *string, to, reason string, changedAt time.Time) error
		Get(ctx context.Context, id uuid.UUID) (model.Trip, error)
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)

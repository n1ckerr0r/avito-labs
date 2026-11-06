//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type (
	tripFinisher interface {
		Execute(ctx context.Context, id uuid.UUID) (model.Trip, error)
	}

	metrics interface {
		IncCompleted(ctx context.Context)
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
	}
)

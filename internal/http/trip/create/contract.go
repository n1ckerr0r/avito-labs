//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"

	"github.com/n1ckerr0r/avito-labs/internal/model"
	usecase "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
)

type (
	tripCreator interface {
		Execute(ctx context.Context, in usecase.Input) (result model.Trip, created bool, err error)
	}

	metrics interface {
		IncCreated(ctx context.Context)
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
	}
)

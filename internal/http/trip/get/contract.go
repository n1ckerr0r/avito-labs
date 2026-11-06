//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test
package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type tripGetter interface {
	Execute(ctx context.Context, id uuid.UUID) (model.Trip, error)
}

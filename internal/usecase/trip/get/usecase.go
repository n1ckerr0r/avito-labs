// Package get implements the "get trip" usecase.
package get

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type Usecase struct {
	repo    tripRepository
	timeout time.Duration
}

func New(repo tripRepository, timeout time.Duration) *Usecase {
	return &Usecase{repo: repo, timeout: timeout}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	return u.repo.Get(ctx, id)
}

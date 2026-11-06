// Package create implements the "create trip" usecase.
package create

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type Input struct {
	UserID, DriverID     uuid.UUID
	StartPoint, EndPoint trip.Coordinates
	Price                int64
	IdempotencyKey       *uuid.UUID
	RequestHash          []byte
}

type Usecase struct {
	repo tripRepository
	tx   txManager
}

func New(repo tripRepository, tx txManager) *Usecase {
	return &Usecase{repo: repo, tx: tx}
}

// Execute creates a trip, or returns the one created by a matching idempotency key.
func (u *Usecase) Execute(ctx context.Context, in Input) (result model.Trip, created bool, err error) {
	t, err := trip.NewTrip(uuid.New(), in.UserID, in.DriverID, in.StartPoint, in.EndPoint, in.Price, time.Now().UTC())
	if err != nil {
		return model.Trip{}, false, err
	}

	m := toModel(t)
	result = m

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if in.IdempotencyKey != nil {
			claimed, existingHash, existingTripID, err := u.repo.ClaimIdempotencyKey(ctx, *in.IdempotencyKey, in.RequestHash)
			if err != nil {
				return err
			}

			if !claimed {
				if !bytes.Equal(existingHash, in.RequestHash) {
					return trip.ErrIdempotencyConflict
				}

				result, err = u.repo.Get(ctx, existingTripID)

				return err
			}
		}

		if err := u.repo.Create(ctx, m); err != nil {
			return err
		}

		if err := u.repo.AddStatusHistory(ctx, m.ID, nil, string(trip.StatusActive), "trip created", m.StartedAt); err != nil {
			return err
		}

		if in.IdempotencyKey != nil {
			if err := u.repo.CompleteIdempotencyKey(ctx, *in.IdempotencyKey, m.ID); err != nil {
				return err
			}
		}

		created = true

		return nil
	})

	return result, created, err
}

func toModel(t trip.Trip) model.Trip {
	return model.Trip{
		ID:         t.ID,
		UserID:     t.UserID,
		DriverID:   t.DriverID,
		StartPoint: model.Coordinates(t.StartPoint),
		EndPoint:   model.Coordinates(t.EndPoint),
		Price:      t.Price,
		Status:     string(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}

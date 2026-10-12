package service

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/n1ckerr0r/avito-labs/internal/database"
	"github.com/n1ckerr0r/avito-labs/internal/domain"
	"github.com/n1ckerr0r/avito-labs/internal/repository"
)

type Trips struct {
	repo *repository.Trips
	tx   database.TxManager
}

func NewTrips(repo *repository.Trips, tx database.TxManager) *Trips {
	return &Trips{repo: repo, tx: tx}
}

func (s *Trips) Create(ctx context.Context, trip domain.Trip, key *uuid.UUID, requestHash []byte) (result domain.Trip, created bool, err error) {
	result = trip
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if key != nil {
			claimed, existingHash, existingTripID, err := s.repo.ClaimIdempotencyKey(ctx, *key, requestHash)
			if err != nil {
				return err
			}

			if !claimed {
				if !bytes.Equal(existingHash, requestHash) {
					return domain.ErrIdempotencyConflict
				}

				result, err = s.repo.Get(ctx, existingTripID)

				return err
			}
		}

		if err := s.repo.Create(ctx, trip); err != nil {
			return err
		}

		if err := s.repo.AddStatusHistory(ctx, trip.ID, nil, "active", "trip created", trip.StartedAt); err != nil {
			return err
		}

		if key != nil {
			if err := s.repo.CompleteIdempotencyKey(ctx, *key, trip.ID); err != nil {
				return err
			}
		}

		created = true

		return nil
	})

	return result, created, err
}
func (s *Trips) Get(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.repo.Get(ctx, id)
}
func (s *Trips) Finish(ctx context.Context, id uuid.UUID) (trip domain.Trip, err error) {
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		trip, err = s.repo.Finish(ctx, id, time.Now().UTC())
		if err != nil {
			return err
		}

		from := "active"

		return s.repo.AddStatusHistory(ctx, id, &from, "completed", "trip finished", *trip.FinishedAt)
	})

	return trip, err
}

// Package finish implements the "finish trip" usecase.
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type Usecase struct {
	repo tripRepository
	tx   txManager
}

func New(repo tripRepository, tx txManager) *Usecase {
	return &Usecase{repo: repo, tx: tx}
}

// Execute locks the trip row, applies the domain transition, and persists it.
func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (result model.Trip, err error) {
	err = u.tx.Do(ctx, func(ctx context.Context) error {
		m, err := u.repo.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}

		t := fromModel(m)
		if err := t.Finish(time.Now().UTC()); err != nil {
			return err
		}

		result = toModel(t)

		if err := u.repo.Update(ctx, result); err != nil {
			return err
		}

		from := string(trip.StatusActive)

		return u.repo.AddStatusHistory(ctx, id, &from, string(trip.StatusCompleted), "trip finished", *result.FinishedAt)
	})

	return result, err
}

func fromModel(m model.Trip) trip.Trip {
	return trip.Trip{
		ID:         m.ID,
		UserID:     m.UserID,
		DriverID:   m.DriverID,
		StartPoint: trip.Coordinates(m.StartPoint),
		EndPoint:   trip.Coordinates(m.EndPoint),
		Price:      m.Price,
		Status:     trip.Status(m.Status),
		StartedAt:  m.StartedAt,
		FinishedAt: m.FinishedAt,
	}
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

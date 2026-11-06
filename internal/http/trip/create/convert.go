package create

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	usecase "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
)

type coordinatesInput struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type requestBody struct {
	UserID     *uuid.UUID        `json:"user_id"`
	DriverID   *uuid.UUID        `json:"driver_id"`
	StartPoint *coordinatesInput `json:"start_point"`
	EndPoint   *coordinatesInput `json:"end_point"`
	Price      *int64            `json:"price"`
}

// decodeRequest checks the request shape only; business rules are validated in the domain.
func decodeRequest(r *http.Request) (requestBody, []byte, error) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return requestBody{}, nil, errors.New("Content-Type must be application/json")
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return requestBody{}, nil, err
	}

	var body requestBody

	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()

	if err := d.Decode(&body); err != nil {
		return requestBody{}, nil, err
	}

	if d.Decode(&struct{}{}) != io.EOF {
		return requestBody{}, nil, errors.New("request body must contain one JSON value")
	}

	if body.UserID == nil || body.DriverID == nil || body.StartPoint == nil || body.EndPoint == nil || body.Price == nil {
		return requestBody{}, nil, errors.New("user_id, driver_id, start_point, end_point, and price are required")
	}

	if body.StartPoint.Latitude == nil || body.StartPoint.Longitude == nil || body.EndPoint.Latitude == nil || body.EndPoint.Longitude == nil {
		return requestBody{}, nil, errors.New("latitude and longitude are required for both points")
	}

	hash := sha256.Sum256(raw)

	return body, hash[:], nil
}

func toInput(body requestBody, idempotencyKey *uuid.UUID, requestHash []byte) usecase.Input {
	return usecase.Input{
		UserID:         *body.UserID,
		DriverID:       *body.DriverID,
		StartPoint:     trip.Coordinates{Latitude: *body.StartPoint.Latitude, Longitude: *body.StartPoint.Longitude},
		EndPoint:       trip.Coordinates{Latitude: *body.EndPoint.Latitude, Longitude: *body.EndPoint.Longitude},
		Price:          *body.Price,
		IdempotencyKey: idempotencyKey,
		RequestHash:    requestHash,
	}
}

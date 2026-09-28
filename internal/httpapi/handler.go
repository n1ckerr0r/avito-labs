package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n1ckerr0r/avito-labs/api"
	"github.com/n1ckerr0r/avito-labs/internal/domain"
	"github.com/n1ckerr0r/avito-labs/internal/service"
)

type Handler struct {
	api.Unimplemented
	trips        *service.Trips
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func New(trips *service.Trips, pool *pgxpool.Pool, queryTimeout time.Duration) *Handler {
	return &Handler{trips: trips, pool: pool, queryTimeout: queryTimeout}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var input api.TripData
	if err := decodeJSON(r, &input); err != nil {
		h.problem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}
	if err := validate(input); err != nil {
		h.problem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}
	now := time.Now().UTC()
	trip := domain.Trip{ID: uuid.New(), UserID: input.UserId, DriverID: input.DriverId, StartPoint: coords(input.StartPoint), EndPoint: coords(input.EndPoint), Price: input.Price, Status: "active", StartedAt: now}
	payload, _ := json.Marshal(input)
	hash := sha256.Sum256(payload)
	createdTrip, created, err := h.trips.Create(r.Context(), trip, params.IdempotencyKey, hash[:])
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if created {
		w.Header().Set("Location", "/api/v1/trips/"+createdTrip.ID.String())
		writeJSON(w, http.StatusCreated, toAPI(createdTrip))
		return
	}
	writeJSON(w, http.StatusOK, toAPI(createdTrip))
}
func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	trip, err := h.trips.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPI(trip))
}
func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	trip, err := h.trips.Finish(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPI(trip))
}
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		h.problem(w, r, 404, "trip_not_found", "Trip not found", "Trip not found")
	case errors.Is(err, domain.ErrCompleted):
		h.problem(w, r, 409, "trip_completed", "Trip completed", "Trip is already completed")
	case errors.Is(err, domain.ErrDriverBusy):
		h.problem(w, r, 409, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, domain.ErrIdempotencyConflict):
		h.problem(w, r, 409, "idempotency_conflict", "Idempotency conflict", "Idempotency key was used with a different request")
	default:
		h.problem(w, r, 500, "internal_error", "Internal server error", "An unexpected error occurred")
	}
}
func (h *Handler) problem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{Type: "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"), Title: title, Status: int32(status), Detail: &detail, Instance: ptr(r.URL.Path), Code: code})
}
func ErrorHandler(w http.ResponseWriter, r *http.Request, _ error) {
	(&Handler{}).problem(w, r, 400, "invalid_request", "Invalid request", "Invalid request parameter")
}
func decodeJSON(r *http.Request, target any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("Content-Type must be application/json")
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}
func validate(t api.TripData) error {
	if t.UserId == uuid.Nil || t.DriverId == uuid.Nil {
		return errors.New("user_id and driver_id must be non-empty UUIDs")
	}
	if t.Price < 0 {
		return errors.New("price must not be negative")
	}
	if t.StartPoint.Latitude < -90 || t.StartPoint.Latitude > 90 || t.EndPoint.Latitude < -90 || t.EndPoint.Latitude > 90 || t.StartPoint.Longitude < -180 || t.StartPoint.Longitude > 180 || t.EndPoint.Longitude < -180 || t.EndPoint.Longitude > 180 {
		return errors.New("coordinates are outside valid ranges")
	}
	return nil
}
func coords(c api.Coordinates) domain.Coordinates {
	return domain.Coordinates{Latitude: c.Latitude, Longitude: c.Longitude}
}
func toAPI(t domain.Trip) api.Trip {
	return api.Trip{Id: t.ID, UserId: t.UserID, DriverId: t.DriverID, StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude}, EndPoint: api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude}, Price: t.Price, Status: api.TripStatus(t.Status), StartedAt: t.StartedAt, FinishedAt: t.FinishedAt}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func ptr(s string) *string { return &s }

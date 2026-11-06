// Package finish implements the HTTP handler for finishing a trip.
package finish

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/http/domainerror"
	"github.com/n1ckerr0r/avito-labs/internal/http/httputil"
)

type Handler struct {
	usecase tripFinisher
	metrics metrics
	logger  logger
}

func New(usecase tripFinisher, metrics metrics, logger logger) *Handler {
	return &Handler{usecase: usecase, metrics: metrics, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	trip, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		status, problem := domainerror.Problem(err)
		httputil.WriteProblem(w, r, status, problem)

		return
	}

	h.metrics.IncCompleted(r.Context())
	h.logger.InfoContext(r.Context(), "trip finished", "trip_id", trip.ID)
	httputil.WriteJSON(w, http.StatusOK, httputil.ToAPI(trip))
}

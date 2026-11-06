// Package get implements the HTTP handler for fetching a trip by id.
package get

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/n1ckerr0r/avito-labs/internal/http/domainerror"
	"github.com/n1ckerr0r/avito-labs/internal/http/httputil"
)

type Handler struct {
	usecase tripGetter
}

func New(usecase tripGetter) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	trip, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		status, problem := domainerror.Problem(err)
		httputil.WriteProblem(w, r, status, problem)

		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.ToAPI(trip))
}

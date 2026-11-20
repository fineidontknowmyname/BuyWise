package decision

import (
	"errors"
	"log/slog"
	"net/http"
	"uuid"

	"selfemi/internal/platform/httpjson"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /goals/{id}/decision", h.evaluate)
	mux.HandleFunc("POST /goals/{id}/decision", h.evaluate)
	mux.HandleFunc("GET /goals/{id}/decision/history", h.history)
}

func (h *Handler) evaluate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, ErrGoalNotFound)
		return
	}
	save := r.Method == http.MethodPost
	value, err := h.service.Evaluate(r.Context(), id, save)
	if err != nil {
		h.fail(w, err)
		return
	}
	status := http.StatusOK
	if save {
		status = http.StatusCreated
	}
	httpjson.Write(w, status, value)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, ErrGoalNotFound)
		return
	}
	values, err := h.service.History(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, values)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrGoalNotFound):
		httpjson.WriteError(w, 404, "goal_not_found", "No goal with that ID.")
	case errors.Is(err, ErrHistoryUnavailable):
		httpjson.WriteError(w, 503, "decision_history_unavailable", "Decision history is temporarily unavailable.")
	default:
		h.logger.Error("purchase decision request failed", "error", err)
		httpjson.WriteError(w, 500, "internal_error", "Something went wrong.")
	}
}

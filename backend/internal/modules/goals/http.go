package goals

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
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

type goalResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type createRequest struct {
	Name string `json:"name"`
}

func toResponse(goal Goal) goalResponse {
	return goalResponse{ID: goal.ID.String(), Name: goal.Name, CreatedAt: goal.CreatedAt}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /goals", h.create)
	mux.HandleFunc("GET /goals", h.list)
	mux.HandleFunc("GET /goals/{id}", h.get)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "The request body must be JSON with a name.")
		return
	}
	goal, err := h.service.Create(r.Context(), req.Name)
	if errors.Is(err, ErrInvalidName) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_name", "The goal name must be 1 to 200 characters.")
		return
	}
	if err != nil {
		h.internalError(w, "create goal", err)
		return
	}
	httpjson.Write(w, http.StatusCreated, toResponse(goal))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.List(r.Context())
	if err != nil {
		h.internalError(w, "list goals", err)
		return
	}
	out := make([]goalResponse, 0, len(list))
	for _, goal := range list {
		out = append(out, toResponse(goal))
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "goal_not_found", "No goal with that ID.")
		return
	}
	goal, err := h.service.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "goal_not_found", "No goal with that ID.")
		return
	}
	if err != nil {
		h.internalError(w, "get goal", err)
		return
	}
	httpjson.Write(w, http.StatusOK, toResponse(goal))
}

func (h *Handler) internalError(w http.ResponseWriter, action string, err error) {
	h.logger.Error(action+" failed", "error", err)
	httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Something went wrong.")
}

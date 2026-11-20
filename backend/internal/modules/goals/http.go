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
	LegacySavedPaise int64     `json:"legacySavedPaise,omitempty"`
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	TargetPricePaise int64     `json:"targetPricePaise"`
	SavedAmountPaise int64     `json:"savedAmountPaise"`
	TargetDate       *string   `json:"targetDate,omitempty"`
	Priority         string    `json:"priority"`
	ProductID        *string   `json:"productId,omitempty"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
}

type createRequest struct {
	LegacySavedPaise int64   `json:"legacySavedPaise,omitempty"`
	Name             string  `json:"name"`
	TargetPricePaise int64   `json:"targetPricePaise,omitempty"`
	SavedAmountPaise int64   `json:"savedAmountPaise,omitempty"`
	TargetDate       *string `json:"targetDate,omitempty"`
	Priority         string  `json:"priority,omitempty"`
	ProductID        *string `json:"productId,omitempty"`
	Status           string  `json:"status,omitempty"`
}

func toResponse(goal Goal) goalResponse {
	var prodID *string
	if goal.ProductID != nil {
		str := goal.ProductID.String()
		prodID = &str
	}
	allocated := goal.AllocatedPaise
	if goal.Status == "cancelled" || goal.Status == "completed" {
		allocated = 0
	}
	return goalResponse{LegacySavedPaise: goal.SavedAmountPaise,
		ID:               goal.ID.String(),
		Name:             goal.Name,
		TargetPricePaise: goal.TargetPricePaise,
		SavedAmountPaise: allocated,
		TargetDate:       goal.TargetDate,
		Priority:         goal.Priority,
		ProductID:        prodID,
		Status:           goal.Status,
		CreatedAt:        goal.CreatedAt,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /goals", h.create)
	mux.HandleFunc("GET /goals", h.list)
	mux.HandleFunc("GET /goals/{id}", h.get)
	mux.HandleFunc("PUT /goals/{id}", h.update)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "The request body must be JSON with a name.")
		return
	}
	var prodUUID *uuid.UUID
	if req.ProductID != nil && *req.ProductID != "" {
		parsed, err := uuid.Parse(*req.ProductID)
		if err == nil {
			prodUUID = &parsed
		}
	}

	g := Goal{
		Name:             req.Name,
		TargetPricePaise: req.TargetPricePaise,
		TargetDate:       req.TargetDate,
		Priority:         req.Priority,
		ProductID:        prodUUID,
		Status:           req.Status,
	}

	goal, err := h.service.CreateWithDetails(r.Context(), g)
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

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "goal_not_found", "No goal with that ID.")
		return
	}
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid goal payload.")
		return
	}
	var prodUUID *uuid.UUID
	if req.ProductID != nil && *req.ProductID != "" {
		parsed, err := uuid.Parse(*req.ProductID)
		if err == nil {
			prodUUID = &parsed
		}
	}

	g := Goal{
		ID:               id,
		Name:             req.Name,
		TargetPricePaise: req.TargetPricePaise,
		TargetDate:       req.TargetDate,
		Priority:         req.Priority,
		ProductID:        prodUUID,
		Status:           req.Status,
	}

	_, err = h.service.Update(r.Context(), g)
	if errors.Is(err, ErrNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "goal_not_found", "No goal with that ID.")
		return
	}
	if errors.Is(err, ErrInvalidName) {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_name", "The goal name must be 1 to 200 characters.")
		return
	}
	if err != nil {
		h.internalError(w, "update goal", err)
		return
	}
	updated, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.internalError(w, "read updated goal", err)
		return
	}
	httpjson.Write(w, http.StatusOK, toResponse(updated))
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

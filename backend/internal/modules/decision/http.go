package decision

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"selfemi/internal/platform/httpjson"
	"selfemi/internal/platform/provenance"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type decisionResponse struct {
	GoalID        string                `json:"goalId"`
	GoalName      string                `json:"goalName"`
	State         State                 `json:"state"`
	ReasonCodes   []string              `json:"reasonCodes"`
	Inputs        []provenance.Labelled `json:"inputs"`
	RulesVersion  string                `json:"rulesVersion"`
	DecidedAt     time.Time             `json:"decidedAt"`
	AllInputsReal bool                  `json:"allInputsReal"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /goals/{id}/decision", h.evaluate)
}

func (h *Handler) evaluate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, http.StatusNotFound, "goal_not_found", "No goal with that ID.")
		return
	}
	d, err := h.service.Evaluate(r.Context(), id)
	if errors.Is(err, ErrGoalNotFound) {
		httpjson.WriteError(w, http.StatusNotFound, "goal_not_found", "No goal with that ID.")
		return
	}
	if err != nil {
		h.logger.Error("evaluate decision failed", "error", err)
		httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Something went wrong.")
		return
	}
	httpjson.Write(w, http.StatusOK, decisionResponse{
		GoalID:        d.GoalID.String(),
		GoalName:      d.GoalName,
		State:         d.State,
		ReasonCodes:   d.ReasonCodes,
		Inputs:        d.Inputs,
		RulesVersion:  d.RulesVersion,
		DecidedAt:     d.DecidedAt,
		AllInputsReal: d.AllInputsReal(),
	})
}

package goals

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"selfemi/internal/platform/httpjson"
)

type Planner struct {
	store   *Store
	finance PlanningFinance
	logger  *slog.Logger
	now     func() time.Time
}

func NewPlanner(store *Store, finance PlanningFinance, logger *slog.Logger, now func() time.Time) *Planner {
	return &Planner{store: store, finance: finance, logger: logger, now: now}
}

type fundingAccount struct {
	PlanningAccount
	ReservedPaise  int64 `json:"reservedPaise"`
	AvailablePaise int64 `json:"availablePaise"`
}

type planView struct {
	Goal          goalResponse     `json:"goal"`
	Terms         planTerms        `json:"terms"`
	CapacityPaise int64            `json:"capacityPaise"`
	ProfileReady  bool             `json:"profileReady"`
	FundedPaise   int64            `json:"fundedPaise"`
	BackingAtRisk bool             `json:"backingAtRisk"`
	Projection    goalProjection   `json:"projection"`
	Accounts      []fundingAccount `json:"accounts"`
	Contributions []fundingEntry   `json:"contributions"`
	History       []planTerms      `json:"history"`
}

func (p *Planner) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /goals/{id}/plan", p.route(false, p.read))
	mux.HandleFunc("PUT /goals/{id}/plan", p.route(true, p.save))
	mux.HandleFunc("POST /goals/{id}/plan/preview", p.route(false, p.preview))
	mux.HandleFunc("POST /goals/{id}/contributions", p.route(true, p.fund))
	mux.HandleFunc("PUT /goals/{id}/lifecycle", p.route(true, p.lifecycle))
}

func (p *Planner) route(write bool, run func(http.ResponseWriter, *http.Request, Goal) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			p.fail(w, ErrNotFound)
			return
		}
		if write {
			if err := p.store.lockPlanning(r.Context()); err != nil {
				p.fail(w, err)
				return
			}
		}
		goal, err := p.store.Get(r.Context(), id)
		if err == nil {
			err = run(w, r, goal)
		}
		if err != nil {
			p.fail(w, err)
		}
	}
}

func (p *Planner) view(r *http.Request, goal Goal, proposed *planTerms) (planView, error) {
	ctx := r.Context()
	total, configured, accounts, err := p.finance.Snapshot(ctx)
	if err != nil {
		return planView{}, err
	}
	candidates, err := p.store.planCandidates(ctx)
	if err != nil {
		return planView{}, err
	}
	reservations, err := p.store.reservations(ctx)
	if err != nil {
		return planView{}, err
	}
	out := planView{Goal: toResponse(goal), CapacityPaise: total, ProfileReady: configured, Accounts: []fundingAccount{}}
	for _, account := range accounts {
		reserved := reservations[account.ID]
		available := max(0, account.BalancePaise-reserved)
		if reserved > 0 && (account.BalancePaise < reserved || account.Purpose == "emergency_reserve") {
			out.BackingAtRisk = true
		}
		if account.Purpose == "emergency_reserve" {
			available = 0
		}
		out.Accounts = append(out.Accounts, fundingAccount{account, reserved, available})
	}
	orderCandidates(candidates)
	available := total
	today := planningDate(p.now())
	for _, candidate := range candidates {
		if candidate.Goal.ID == goal.ID {
			out.Terms, out.FundedPaise = candidate.Terms, candidate.FundedPaise
			if proposed != nil {
				candidate.Terms = *proposed
				if candidate.Goal.Status == "paused" {
					candidate.Goal.Status = "active"
				}
			}
			out.Projection = project(candidate, available, configured, today)
			return out, nil
		}
		available -= project(candidate, available, configured, today).MonthlyAllocatedPaise
	}
	return planView{}, ErrNotFound
}

func (p *Planner) read(w http.ResponseWriter, r *http.Request, goal Goal) error {
	out, err := p.view(r, goal, nil)
	if err != nil {
		return err
	}
	out.Contributions, err = p.store.fundingHistory(r.Context(), goal.ID)
	if err != nil {
		return err
	}
	out.History, err = p.store.planHistory(r.Context(), goal.ID)
	if err != nil {
		return err
	}
	httpjson.Write(w, 200, out)
	return nil
}

func (p *Planner) preview(w http.ResponseWriter, r *http.Request, goal Goal) error {
	var terms planTerms
	if httpjson.Decode(w, r, &terms) != nil || !terms.valid() {
		httpjson.WriteError(w, 400, "invalid_plan", "Choose a valid mode, frequency and contribution.")
		return nil
	}
	out, err := p.view(r, goal, &terms)
	if err != nil {
		return err
	}
	httpjson.Write(w, 200, out.Projection)
	return nil
}

func (p *Planner) save(w http.ResponseWriter, r *http.Request, goal Goal) error {
	var terms planTerms
	if httpjson.Decode(w, r, &terms) != nil || !terms.valid() {
		httpjson.WriteError(w, 400, "invalid_plan", "Choose a valid mode, frequency and contribution.")
		return nil
	}
	if goal.Status != "active" && goal.Status != "paused" {
		return errPlanUnavailable
	}
	plan, err := p.view(r, goal, &terms)
	if err != nil {
		return err
	}
	if plan.Terms.Revision != terms.Revision {
		return errPlanConflict
	}
	if terms.Mode == "intelligent" && terms.ApprovedPaise > 0 &&
		(!plan.ProfileReady || terms.ApprovedPaise > plan.Projection.RecommendedPaise) {
		return errContributionCapacity
	}
	if err := p.store.saveTerms(r.Context(), goal.ID, terms); err != nil {
		return err
	}
	terms.Revision++
	httpjson.Write(w, 200, terms)
	return nil
}

func (p *Planner) fund(w http.ResponseWriter, r *http.Request, goal Goal) error {
	var input fundingInput
	if httpjson.Decode(w, r, &input) != nil || !input.valid() {
		httpjson.WriteError(w, 400, "invalid_funding", "Choose an account and enter a positive amount.")
		return nil
	}
	repeated, err := p.store.priorFunding(r.Context(), goal.ID, input)
	if err != nil {
		return err
	}
	if repeated {
		w.WriteHeader(204)
		return nil
	}
	if goal.Status != "active" {
		return errPlanUnavailable
	}
	plan, err := p.view(r, goal, nil)
	if err != nil {
		return err
	}
	if input.AmountPaise > plan.Projection.RemainingPaise {
		return errFundingConflict
	}
	for _, account := range plan.Accounts {
		if account.ID == input.AccountID && input.AmountPaise <= account.AvailablePaise {
			if err := p.store.addFunding(r.Context(), goal.ID, input); err != nil {
				return err
			}
			w.WriteHeader(204)
			return nil
		}
	}
	return errPlanningAccount
}

func (p *Planner) lifecycle(w http.ResponseWriter, r *http.Request, goal Goal) error {
	var input struct {
		Status string `json:"status"`
	}
	if httpjson.Decode(w, r, &input) != nil || (input.Status != "active" && input.Status != "paused" && input.Status != "cancelled") {
		httpjson.WriteError(w, 400, "invalid_status", "Choose active, paused, or cancelled.")
		return nil
	}
	if err := p.store.setLifecycle(r.Context(), goal.ID, input.Status); err != nil {
		return err
	}
	w.WriteHeader(204)
	return nil
}

func (p *Planner) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpjson.WriteError(w, 404, "goal_not_found", "No goal with that ID.")
	case errors.Is(err, errPlanConflict):
		httpjson.WriteError(w, 409, "plan_conflict", "This plan changed. Reload before saving again.")
	case errors.Is(err, errFundingConflict):
		httpjson.WriteError(w, 409, "funding_conflict", "This allocation exceeds the goal's remaining amount or conflicts with an earlier request.")
	case errors.Is(err, errPlanUnavailable):
		httpjson.WriteError(w, 409, "goal_inactive", "This action is unavailable for the current goal status.")
	case errors.Is(err, errContributionCapacity):
		httpjson.WriteError(w, 400, "contribution_capacity", "Enter a contribution within the recommended amount for this goal.")
	case errors.Is(err, errPlanningAccount):
		httpjson.WriteError(w, 400, "account_unavailable", "Choose an eligible account with enough unallocated funds.")
	default:
		p.logger.Error("goal planning request failed", "error", err)
		httpjson.WriteError(w, 500, "internal_error", "Something went wrong.")
	}
}

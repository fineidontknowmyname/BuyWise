package finance

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"selfemi/internal/platform/httpjson"
)

type foundationHandler struct {
	store  *Store
	logger *slog.Logger
	today  func() time.Time
}

type foundationView struct {
	Profile               financialProfile    `json:"profile"`
	Capacity              capacity            `json:"capacity"`
	Accounts              []designatedAccount `json:"accounts"`
	EmergencySavingsPaise int64               `json:"emergencySavingsPaise"`
	GeneralSavingsPaise   int64               `json:"generalSavingsPaise"`
	EmergencyGapPaise     int64               `json:"emergencyGapPaise"`
}

type designatedAccount struct {
	accountResponse
	Purpose string `json:"purpose"`
}

func (h foundationHandler) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /financial-profile", h.getProfile)
	mux.HandleFunc("PUT /financial-profile", h.putProfile)
	mux.HandleFunc("PUT /accounts/{id}/purpose", h.putPurpose)
	mux.HandleFunc("GET /transfers", h.transfers)
	mux.HandleFunc("POST /transfers", h.postTransfer)
	mux.HandleFunc("POST /transfers/{id}/reverse", h.reverse)
}

func (h foundationHandler) getProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile, err := h.store.loadProfile(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	accounts, err := h.store.ListAccounts(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	purposes, err := h.store.designations(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	view := foundationView{Profile: profile, Capacity: profile.estimate(), Accounts: []designatedAccount{}}
	for _, a := range accounts {
		purpose := purposes[a.ID.String()]
		switch purpose {
		case "emergency_reserve":
			view.EmergencySavingsPaise += max(0, a.BalancePaise)
		case "general_savings":
			view.GeneralSavingsPaise += max(0, a.BalancePaise)
		default:
			purpose = "everyday"
		}
		view.Accounts = append(view.Accounts, designatedAccount{accountResponse: toAccountResponse(a), Purpose: purpose})
	}
	view.EmergencyGapPaise = max(0, profile.EmergencyTargetPaise-view.EmergencySavingsPaise)
	httpjson.Write(w, 200, view)
}

func (h foundationHandler) putProfile(w http.ResponseWriter, r *http.Request) {
	var profile financialProfile
	if httpjson.Decode(w, r, &profile) != nil || !profile.validate() {
		httpjson.WriteError(w, 400, "invalid_profile", "Check the financial assumptions, amount and frequency of each entry.")
		return
	}
	profile, err := h.store.saveProfile(r.Context(), profile)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, 200, profile)
}

func (h foundationHandler) putPurpose(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, ErrAccountNotFound)
		return
	}
	var input struct {
		Purpose string `json:"purpose"`
	}
	if httpjson.Decode(w, r, &input) != nil ||
		(input.Purpose != "everyday" && input.Purpose != "general_savings" && input.Purpose != "emergency_reserve") {
		httpjson.WriteError(w, 400, "invalid_purpose", "Choose a valid account purpose.")
		return
	}
	if err := h.store.designate(r.Context(), id, input.Purpose); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h foundationHandler) transfers(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.listTransfers(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, 200, items)
}

func (h foundationHandler) postTransfer(w http.ResponseWriter, r *http.Request) {
	var input transferInput
	if httpjson.Decode(w, r, &input) != nil || !input.validate(h.today()) {
		httpjson.WriteError(w, 400, "invalid_transfer", "Choose two different accounts and enter a valid amount and date.")
		return
	}
	item, created, err := h.store.createTransfer(r.Context(), input)
	if err != nil {
		h.fail(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpjson.Write(w, status, item)
}

func (h foundationHandler) reverse(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, errTransferNotFound)
		return
	}
	item, err := h.store.reverseTransfer(r.Context(), id, h.today())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, 200, item)
}

func (h foundationHandler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		httpjson.WriteError(w, 404, "account_not_found", "Account not found.")
	case errors.Is(err, errTransferNotFound):
		httpjson.WriteError(w, 404, "transfer_not_found", "Transfer not found or cannot be reversed.")
	case errors.Is(err, errProfileConflict):
		httpjson.WriteError(w, 409, "profile_conflict", "This profile changed. Reload before saving again.")
	case errors.Is(err, errTransferConflict):
		httpjson.WriteError(w, 409, "transfer_conflict", "This request was already used for a different transfer.")
	default:
		h.logger.Error("financial foundation request failed", "error", err)
		httpjson.WriteError(w, 500, "internal_error", "Something went wrong.")
	}
}

package finance

import (
	"net/http"
	"strings"

	"selfemi/internal/platform/httpjson"
)

func reservedCaptureKey(key string) bool {
	return strings.HasPrefix(key, captureKeyPrefix) || strings.HasPrefix(key, "expense-import-")
}

func (h *Handler) recordExpense(w http.ResponseWriter, r *http.Request) {
	var fields expenseFields
	if httpjson.Decode(w, r, &fields) != nil {
		h.badBody(w)
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if !validKey(key) || reservedCaptureKey(key) {
		h.fail(w, "record expense", ErrInvalidKey)
		return
	}
	input, err := fields.transaction(h.service.Today(), key)
	if err != nil {
		h.fail(w, "record expense", err)
		return
	}
	tx, created, err := h.service.captureExpense(r.Context(), input)
	if err != nil {
		h.fail(w, "record expense", err)
		return
	}
	writeCapture(w, created, toTransactionResponse(tx))
}

func writeCapture(w http.ResponseWriter, created bool, value any) {
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpjson.Write(w, status, value)
}

func (h *Handler) registerCapture(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /expenses/candidates", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		items, err := store.listCandidates(ctx)
		if err != nil {
			h.fail(w, "list expense suggestions", err)
			return
		}
		if len(items) > 0 {
			categories, err := h.service.ListCategories(ctx)
			if err != nil {
				h.fail(w, "suggest expense category", err)
				return
			}
			for i := range items {
				items[i].Category = suggestExpenseCategory(items[i].Merchant, categories)
			}
		}
		httpjson.Write(w, http.StatusOK, items)
	})
	mux.HandleFunc("POST /expenses/candidates/notifications", func(w http.ResponseWriter, r *http.Request) {
		var fields notificationInput
		if httpjson.Decode(w, r, &fields) != nil {
			h.badBody(w)
			return
		}
		item, fingerprint, reference, err := fields.candidate(h.service.now())
		if err != nil {
			h.fail(w, "ingest notification expense", err)
			return
		}
		ctx := r.Context()
		if err := store.lockCapture(ctx); err != nil {
			h.fail(w, "ingest notification expense", err)
			return
		}
		item, created, err := store.ingestCandidate(ctx, item, fingerprint, reference)
		if err != nil {
			h.fail(w, "ingest notification expense", err)
			return
		}
		writeCapture(w, created, item)
	})
	mux.HandleFunc("POST /expenses/candidates/{id}/confirm", func(w http.ResponseWriter, r *http.Request) {
		h.confirmCandidate(w, r, store)
	})
	mux.HandleFunc("POST /expenses/candidates/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
		id, ok := h.pathID(w, r, errCandidateNotFound)
		if !ok {
			return
		}
		ctx := r.Context()
		if err := store.lockCapture(ctx); err != nil {
			h.fail(w, "reject expense suggestion", err)
			return
		}
		item, err := store.candidateByID(ctx, id)
		if err == nil {
			switch item.Status {
			case "pending":
				err = store.reviewCandidate(ctx, id, "rejected", nil, h.service.now())
			case "confirmed":
				err = errCandidateReviewed
			}
		}
		if err != nil {
			h.fail(w, "reject expense suggestion", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func (h *Handler) confirmCandidate(w http.ResponseWriter, r *http.Request, store *Store) {
	id, ok := h.pathID(w, r, errCandidateNotFound)
	if !ok {
		return
	}
	var request struct {
		expenseFields
		Confirmed      bool `json:"confirmed"`
		AllowDuplicate bool `json:"allowDuplicate"`
	}
	if httpjson.Decode(w, r, &request) != nil || !request.Confirmed {
		httpjson.WriteError(w, http.StatusBadRequest, "invalid_confirmation", "Review and confirm the expense details.")
		return
	}
	input, err := request.transaction(h.service.Today(), captureKeyPrefix+id.String())
	if err != nil {
		h.fail(w, "confirm expense suggestion", err)
		return
	}
	ctx := r.Context()
	if err := store.lockCapture(ctx); err != nil {
		h.fail(w, "confirm expense suggestion", err)
		return
	}
	item, err := store.candidateByID(ctx, id)
	if err != nil {
		h.fail(w, "confirm expense suggestion", err)
		return
	}
	if item.Status == "confirmed" {
		prior, err := store.transactionBy(ctx, "id", *item.TransactionID)
		if err != nil {
			h.fail(w, "read confirmed expense", err)
		} else if prior == nil || !matchesExpense(*prior, input) {
			h.fail(w, "confirm expense suggestion", errCandidateReviewed)
		} else {
			writeCapture(w, false, toTransactionResponse(*prior))
		}
		return
	}
	if item.Status != "pending" {
		h.fail(w, "confirm expense suggestion", errCandidateReviewed)
		return
	}
	if !request.AllowDuplicate {
		exists, err := store.expenseExists(ctx, input)
		if err != nil {
			h.fail(w, "check possible duplicate expense", err)
			return
		}
		if exists {
			h.fail(w, "confirm expense suggestion", errPossibleDuplicate)
			return
		}
	}
	tx, _, err := h.service.captureExpense(ctx, input)
	if err == nil {
		transaction := tx.ID.String()
		err = store.reviewCandidate(ctx, id, "confirmed", &transaction, h.service.now())
	}
	if err != nil {
		h.fail(w, "confirm expense suggestion", err)
		return
	}
	writeCapture(w, true, toTransactionResponse(tx))
}

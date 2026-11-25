package finance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/jackc/pgx/v5"
	"selfemi/internal/platform/database"
	"selfemi/internal/platform/httpjson"
)

type importRow struct {
	Index             int      `json:"index"`
	AmountPaise       *int64   `json:"amountPaise"`
	OccurredOn        string   `json:"occurredOn"`
	Description       string   `json:"description"`
	Reference         string   `json:"reference"`
	Kind              string   `json:"kind"`
	Warnings          []string `json:"warnings"`
	PossibleDuplicate bool     `json:"possibleDuplicate"`
	TransactionID     string   `json:"transactionId"`
	AccountID         string   `json:"accountId,omitempty"`
	CategoryID        string   `json:"categoryId,omitempty"`
	Status            string   `json:"status"`
}

type expenseImport struct {
	ID        string      `json:"id"`
	Filename  string      `json:"filename"`
	Format    string      `json:"format"`
	Status    string      `json:"status"`
	Revision  int         `json:"revision"`
	Rows      []importRow `json:"rows"`
	CreatedAt time.Time   `json:"createdAt"`
}

type importSourceRow struct {
	AmountPaise *int64   `json:"amountPaise"`
	OccurredOn  string   `json:"occurredOn"`
	Description string   `json:"description"`
	Reference   string   `json:"reference"`
	Kind        string   `json:"kind"`
	Warnings    []string `json:"warnings"`
}

func normalizeImportRows(source []importSourceRow, today time.Time) ([]importRow, error) {
	if len(source) == 0 || len(source) > 200 {
		return nil, errors.New("Import between 1 and 200 rows at a time.")
	}
	rows := make([]importRow, len(source))
	seen := map[string]bool{}
	for i, in := range source {
		if utf8.RuneCountInString(in.Description) > 500 || utf8.RuneCountInString(in.Reference) > 200 || len(in.Warnings) > 8 || len(in.OccurredOn) > 32 {
			return nil, errors.New("An import row exceeds the supported field limits.")
		}
		row := importRow{Index: i, AmountPaise: in.AmountPaise, OccurredOn: in.OccurredOn, Description: strings.TrimSpace(in.Description), Reference: strings.TrimSpace(in.Reference), Kind: in.Kind, Status: "pending", Warnings: []string{}}
		for _, warning := range in.Warnings {
			if len(warning) > 300 {
				return nil, errors.New("An import warning is too long.")
			}
			row.Warnings = append(row.Warnings, warning)
		}
		if row.Kind != "expense" && row.Kind != "income" && row.Kind != "transfer" && row.Kind != "refund" {
			row.Kind = "unknown"
		}
		if row.Kind != "expense" {
			row.Warnings = append(row.Warnings, "Not identified as spending. Leave credits, refunds and transfers out of expense posting.")
		}
		if row.AmountPaise == nil || *row.AmountPaise < 1 || *row.AmountPaise > MaxAmountPaise {
			row.AmountPaise = nil
			row.Warnings = append(row.Warnings, "Enter a valid positive amount.")
		}
		date, err := time.Parse(DateLayout, row.OccurredOn)
		if err != nil || date.Year() < 1 || date.After(today) {
			row.Warnings = append(row.Warnings, "Correct the date before recording; future dates are not expenses.")
		}
		if row.AmountPaise != nil {
			key := fmt.Sprintf("%s:%d", row.OccurredOn, *row.AmountPaise)
			row.PossibleDuplicate = seen[key]
			seen[key] = true
		}
		rows[i] = row
	}
	return rows, nil
}

const importColumns = `id::text, filename, format, status, revision, rows, created_at`

func scanExpenseImport(row pgx.Row) (expenseImport, error) {
	var item expenseImport
	var raw []byte
	err := row.Scan(&item.ID, &item.Filename, &item.Format, &item.Status, &item.Revision, &raw, &item.CreatedAt)
	if err == nil {
		err = json.Unmarshal(raw, &item.Rows)
	}
	return item, err
}

func (s *Store) getExpenseImport(ctx context.Context, id string) (expenseImport, error) {
	return scanExpenseImport(database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT `+importColumns+` FROM finance.expense_imports WHERE id=$1::uuid`, id))
}

func (s *Store) saveExpenseImport(ctx context.Context, item expenseImport) error {
	raw, err := json.Marshal(item.Rows)
	if err != nil {
		return err
	}
	_, err = database.Scoped(ctx, s.pool).Exec(ctx, `UPDATE finance.expense_imports SET rows=$2, status=$3, revision=revision+1, updated_at=now() WHERE id=$1::uuid`, item.ID, raw, item.Status)
	return err
}

func (h *Handler) importFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		httpjson.WriteError(w, 404, "import_not_found", "That import was not found.")
		return
	}
	h.fail(w, "expense import", err)
}

func (h *Handler) registerExpenseWorkspace(mux *http.ServeMux, store *Store) {
	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, r *http.Request) { h.listExpenseWorkspace(w, r, store) })
	mux.HandleFunc("GET /expenses/imports", func(w http.ResponseWriter, r *http.Request) {
		rows, err := database.Scoped(r.Context(), store.pool).Query(r.Context(), `SELECT `+importColumns+` FROM finance.expense_imports ORDER BY created_at DESC,id DESC LIMIT 20`)
		if err != nil {
			h.importFailure(w, err)
			return
		}
		defer rows.Close()
		items := []expenseImport{}
		for rows.Next() {
			item, err := scanExpenseImport(rows)
			if err != nil {
				h.importFailure(w, err)
				return
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			h.importFailure(w, err)
			return
		}
		httpjson.Write(w, 200, items)
	})
	mux.HandleFunc("GET /expenses/imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			httpjson.WriteError(w, 404, "import_not_found", "That import was not found.")
			return
		}
		item, err := store.getExpenseImport(r.Context(), id.String())
		if err != nil {
			h.importFailure(w, err)
			return
		}
		httpjson.Write(w, 200, item)
	})
	mux.HandleFunc("POST /expenses/imports", func(w http.ResponseWriter, r *http.Request) { h.stageExpenseImport(w, r, store) })
	mux.HandleFunc("POST /expenses/imports/{id}/confirm", func(w http.ResponseWriter, r *http.Request) { h.confirmExpenseImport(w, r, store) })
	mux.HandleFunc("POST /expenses/imports/{id}/reverse", func(w http.ResponseWriter, r *http.Request) { h.finishExpenseImport(w, r, store, true) })
	mux.HandleFunc("POST /expenses/imports/{id}/discard", func(w http.ResponseWriter, r *http.Request) { h.finishExpenseImport(w, r, store, false) })
}

func (h *Handler) stageExpenseImport(w http.ResponseWriter, r *http.Request, store *Store) {
	var input struct {
		Filename    string            `json:"filename"`
		Format      string            `json:"format"`
		Fingerprint string            `json:"fingerprint"`
		Rows        []importSourceRow `json:"rows"`
	}
	if httpjson.Decode(w, r, &input) != nil {
		h.badBody(w)
		return
	}
	if len(input.Filename) == 0 || utf8.RuneCountInString(input.Filename) > 200 || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(input.Fingerprint) || (input.Format != "csv" && input.Format != "pdf" && input.Format != "image") {
		httpjson.WriteError(w, 400, "invalid_import", "Check the file name, format and fingerprint.")
		return
	}
	items, err := normalizeImportRows(input.Rows, h.service.Today())
	if err != nil {
		httpjson.WriteError(w, 400, "invalid_import", err.Error())
		return
	}
	ctx := r.Context()
	if err := store.lockCapture(ctx); err != nil {
		h.importFailure(w, err)
		return
	}
	db := database.Scoped(ctx, store.pool)
	prior, err := scanExpenseImport(db.QueryRow(ctx, `SELECT `+importColumns+` FROM finance.expense_imports WHERE fingerprint=$1`, input.Fingerprint))
	found := err == nil
	if found && prior.Status != "review" && prior.Status != "discarded" {
		httpjson.Write(w, 200, prior)
		return
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.importFailure(w, err)
		return
	}
	for i := range items {
		if items[i].AmountPaise != nil {
			var exists bool
			err = db.QueryRow(ctx, `SELECT EXISTS (SELECT FROM finance.transactions WHERE kind='expense' AND voided_at IS NULL AND occurred_on::text=$1 AND amount_paise=$2)`, items[i].OccurredOn, *items[i].AmountPaise).Scan(&exists)
			if err != nil {
				h.importFailure(w, err)
				return
			}
			items[i].PossibleDuplicate = items[i].PossibleDuplicate || exists
		}
	}
	raw, _ := json.Marshal(items)
	if found {
		previous, _ := json.Marshal(prior.Rows)
		if prior.Status == "discarded" || !bytes.Equal(previous, raw) {
			prior.Rows = items
			prior.Status = "review"
			if err := store.saveExpenseImport(ctx, prior); err != nil {
				h.importFailure(w, err)
				return
			}
			prior.Revision++
		}
		httpjson.Write(w, 200, prior)
		return
	}
	item, err := scanExpenseImport(db.QueryRow(ctx, `INSERT INTO finance.expense_imports (filename,format,fingerprint,rows) VALUES($1,$2,$3,$4) RETURNING `+importColumns, input.Filename, input.Format, input.Fingerprint, raw))
	if err != nil {
		h.importFailure(w, err)
		return
	}
	httpjson.Write(w, 201, item)
}

func (h *Handler) confirmExpenseImport(w http.ResponseWriter, r *http.Request, store *Store) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, 404, "import_not_found", "That import was not found.")
		return
	}
	var input struct {
		Revision  int  `json:"revision"`
		Confirmed bool `json:"confirmed"`
		Rows      []struct {
			Index int `json:"index"`
			expenseFields
			AllowDuplicate bool `json:"allowDuplicate"`
		} `json:"rows"`
	}
	if httpjson.Decode(w, r, &input) != nil {
		h.badBody(w)
		return
	}
	if !input.Confirmed || len(input.Rows) == 0 || len(input.Rows) > 200 {
		httpjson.WriteError(w, 400, "review_required", "Select rows and confirm that you reviewed them.")
		return
	}
	ctx := r.Context()
	if err := store.lockCapture(ctx); err != nil {
		h.importFailure(w, err)
		return
	}
	batch, err := store.getExpenseImport(ctx, id.String())
	if err != nil {
		h.importFailure(w, err)
		return
	}
	if batch.Status == "reversed" || batch.Status == "discarded" {
		httpjson.WriteError(w, 409, "import_closed", "This import is closed.")
		return
	}
	seen := map[int]bool{}
	pending := false
	for _, entry := range input.Rows {
		if entry.Index < 0 || entry.Index >= len(batch.Rows) || seen[entry.Index] {
			httpjson.WriteError(w, 400, "invalid_row", "An import row is invalid or selected twice.")
			return
		}
		seen[entry.Index] = true
		if batch.Rows[entry.Index].Status == "pending" {
			pending = true
		}
	}
	if pending && input.Revision != batch.Revision {
		httpjson.WriteError(w, 409, "import_changed", "This import changed. Reload and review the current rows.")
		return
	}
	for _, entry := range input.Rows {
		row := &batch.Rows[entry.Index]
		txInput, err := entry.transaction(h.service.Today(), "expense-import-"+batch.ID+fmt.Sprintf("-%d", entry.Index))
		if err != nil {
			h.importFailure(w, err)
			return
		}
		if row.Status == "confirmed" {
			prior, err := store.transactionBy(ctx, "id", row.TransactionID)
			if err != nil {
				h.importFailure(w, err)
				return
			}
			if prior == nil || !matchesExpense(*prior, txInput) {
				httpjson.WriteError(w, 409, "import_changed", "An already recorded row was changed. Open its account record to correct it.")
				return
			}
			continue
		}
		if row.Status != "pending" {
			httpjson.WriteError(w, 409, "import_closed", "A selected row is no longer available.")
			return
		}
		duplicate, err := store.expenseExists(ctx, txInput)
		if err != nil {
			h.importFailure(w, err)
			return
		}
		if duplicate && !entry.AllowDuplicate {
			httpjson.WriteError(w, 409, "possible_duplicate", fmt.Sprintf("Row %d matches existing spending. Confirm it is a separate purchase before recording.", entry.Index+1))
			return
		}
		tx, _, err := h.service.captureExpense(ctx, txInput)
		if err != nil {
			h.importFailure(w, err)
			return
		}
		row.TransactionID = tx.ID.String()
		row.AccountID = tx.AccountID.String()
		row.CategoryID = tx.CategoryID.String()
		row.Status = "confirmed"
		row.AmountPaise = &entry.AmountPaise
		row.OccurredOn = entry.OccurredOn
		row.Description = entry.Note
	}
	if pending {
		batch.Status = "confirmed"
		for _, row := range batch.Rows {
			if row.Status == "pending" {
				batch.Status = "partial"
				break
			}
		}
		if err := store.saveExpenseImport(ctx, batch); err != nil {
			h.importFailure(w, err)
			return
		}
		batch.Revision++
	}
	httpjson.Write(w, 200, batch)
}

func (h *Handler) finishExpenseImport(w http.ResponseWriter, r *http.Request, store *Store, reverse bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpjson.WriteError(w, 404, "import_not_found", "That import was not found.")
		return
	}
	var input struct {
		Confirmed bool   `json:"confirmed"`
		Reason    string `json:"reason"`
	}
	if httpjson.Decode(w, r, &input) != nil || !input.Confirmed {
		httpjson.WriteError(w, 400, "review_required", "Confirm this import action.")
		return
	}
	ctx := r.Context()
	if err := store.lockCapture(ctx); err != nil {
		h.importFailure(w, err)
		return
	}
	batch, err := store.getExpenseImport(ctx, id.String())
	if err != nil {
		h.importFailure(w, err)
		return
	}
	if batch.Status == "reversed" || batch.Status == "discarded" {
		httpjson.Write(w, 200, batch)
		return
	}
	if reverse && (strings.TrimSpace(input.Reason) == "" || utf8.RuneCountInString(input.Reason) > 450) {
		httpjson.WriteError(w, 400, "invalid_reason", "Explain why this import is being reversed (up to 500 characters).")
		return
	}
	for i := range batch.Rows {
		row := &batch.Rows[i]
		if row.TransactionID == "" {
			continue
		}
		if !reverse {
			httpjson.WriteError(w, 409, "import_recorded", "Recorded imports must be reversed, not discarded.")
			return
		}
		prior, err := store.transactionBy(ctx, "id", row.TransactionID)
		if err != nil {
			h.importFailure(w, err)
			return
		}
		if prior == nil {
			httpjson.WriteError(w, 409, "missing_transaction", "A linked expense could not be found.")
			return
		}
		if !prior.Voided() {
			if _, err := h.service.VoidTransaction(ctx, prior.ID, "Import reversal: "+input.Reason); err != nil {
				h.importFailure(w, err)
				return
			}
		}
		row.Status = "reversed"
	}
	batch.Status = "discarded"
	if reverse {
		batch.Status = "reversed"
	}
	if err := store.saveExpenseImport(ctx, batch); err != nil {
		h.importFailure(w, err)
		return
	}
	batch.Revision++
	httpjson.Write(w, 200, batch)
}

func (h *Handler) listExpenseWorkspace(w http.ResponseWriter, r *http.Request, store *Store) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = h.service.Today().Format(MonthLayout)
	}
	start, err := time.Parse(MonthLayout, month)
	if err != nil || start.Year() < 1 {
		h.fail(w, "expense month", ErrInvalidMonth)
		return
	}
	account, category, search := r.URL.Query().Get("account"), r.URL.Query().Get("category"), r.URL.Query().Get("search")
	for _, id := range []string{account, category} {
		if id != "" {
			if _, err := uuid.Parse(id); err != nil {
				httpjson.WriteError(w, 400, "invalid_filter", "Choose an existing account or category.")
				return
			}
		}
	}
	if utf8.RuneCountInString(search) > 200 {
		httpjson.WriteError(w, 400, "invalid_filter", "Search is too long.")
		return
	}
	args := []any{start, start.AddDate(0, 1, 0), account, category, search}
	condition := ` FROM finance.transactions WHERE kind='expense' AND voided_at IS NULL AND occurred_on >= $1 AND occurred_on < $2 AND ($3='' OR account_id::text=$3) AND ($4='' OR category_id::text=$4) AND ($5='' OR strpos(lower(note),lower($5))>0)`
	db := database.Scoped(r.Context(), store.pool)
	var total int64
	var count int
	if err := db.QueryRow(r.Context(), `SELECT COALESCE(SUM(amount_paise),0)::bigint,COUNT(*)`+condition, args...).Scan(&total, &count); err != nil {
		h.importFailure(w, err)
		return
	}
	rows, err := db.Query(r.Context(), `SELECT `+transactionColumns+condition+` ORDER BY occurred_on DESC,created_at DESC,id DESC LIMIT 500`, args...)
	if err != nil {
		h.importFailure(w, err)
		return
	}
	defer rows.Close()
	items := []transactionResponse{}
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			h.importFailure(w, err)
			return
		}
		items = append(items, toTransactionResponse(tx))
	}
	if err := rows.Err(); err != nil {
		h.importFailure(w, err)
		return
	}
	httpjson.Write(w, 200, struct {
		Items      []transactionResponse `json:"items"`
		TotalPaise int64                 `json:"totalPaise"`
		Count      int                   `json:"count"`
		Month      string                `json:"month"`
	}{items, total, count, month})
}

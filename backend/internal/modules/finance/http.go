package finance

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

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /accounts", h.createAccount)
	mux.HandleFunc("GET /accounts", h.listAccounts)
	mux.HandleFunc("GET /accounts/{id}", h.getAccount)
	mux.HandleFunc("GET /accounts/{id}/transactions", h.listTransactions)
	mux.HandleFunc("POST /categories", h.createCategory)
	mux.HandleFunc("GET /categories", h.listCategories)
	mux.HandleFunc("GET /categories/{id}", h.getCategory)
	mux.HandleFunc("PUT /categories/{id}", h.updateCategory)
	mux.HandleFunc("PATCH /categories/{id}", h.updateCategory)
	mux.HandleFunc("DELETE /categories/{id}", h.deleteCategory)
	mux.HandleFunc("GET /categories/{id}/transactions/count", h.categoryTransactionCount)
	mux.HandleFunc("POST /transactions", h.createTransaction)
	mux.HandleFunc("POST /transactions/{id}/void", h.voidTransaction)
	mux.HandleFunc("PUT /budgets/{id}", h.setBudget)
	mux.HandleFunc("DELETE /budgets/{id}", h.deleteBudget)
	mux.HandleFunc("GET /budgets", h.budgetReport)
}

type accountResponse struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	OpeningBalancePaise int64     `json:"openingBalancePaise"`
	BalancePaise        int64     `json:"balancePaise"`
	CreatedAt           time.Time `json:"createdAt"`
}

type categoryResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      Kind   `json:"kind"`
	Committed bool   `json:"committed"`
	Archived  bool   `json:"archived"`
}

type transactionResponse struct {
	ID          string     `json:"id"`
	AccountID   string     `json:"accountId"`
	CategoryID  string     `json:"categoryId"`
	Kind        Kind       `json:"kind"`
	AmountPaise int64      `json:"amountPaise"`
	OccurredOn  string     `json:"occurredOn"`
	Note        string     `json:"note"`
	Voided      bool       `json:"voided"`
	VoidedAt    *time.Time `json:"voidedAt"`
	VoidReason  string     `json:"voidReason"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func toAccountResponse(a AccountSummary) accountResponse {
	return accountResponse{
		ID: a.ID.String(), Name: a.Name, OpeningBalancePaise: a.OpeningBalancePaise,
		BalancePaise: a.BalancePaise, CreatedAt: a.CreatedAt,
	}
}

func toCategoryResponse(c Category) categoryResponse {
	return categoryResponse{ID: c.ID.String(), Name: c.Name, Kind: c.Kind, Committed: c.Committed, Archived: c.Archived}
}

func toTransactionResponse(t Transaction) transactionResponse {
	return transactionResponse{
		ID: t.ID.String(), AccountID: t.AccountID.String(), CategoryID: t.CategoryID.String(),
		Kind: t.Kind, AmountPaise: t.AmountPaise, OccurredOn: t.OccurredOn.Format(DateLayout),
		Note: t.Note, Voided: t.Voided(), VoidedAt: t.VoidedAt, VoidReason: t.VoidReason, CreatedAt: t.CreatedAt,
	}
}

type apiProblem struct {
	status  int
	code    string
	message string
}

var problems = map[error]apiProblem{
	ErrInvalidName:         {http.StatusBadRequest, "invalid_name", "The name must be 1 to 200 characters."},
	ErrInvalidAmount:       {http.StatusBadRequest, "invalid_amount", "The amount must be a whole number of paise greater than zero."},
	ErrInvalidOpening:      {http.StatusBadRequest, "invalid_opening_balance", "The opening balance is out of range."},
	ErrInvalidDate:         {http.StatusBadRequest, "invalid_date", "The date must be a real calendar date as YYYY-MM-DD."},
	ErrInvalidKind:         {http.StatusBadRequest, "invalid_kind", "The kind must be income or expense."},
	ErrInvalidNote:         {http.StatusBadRequest, "invalid_note", "The note must be at most 500 characters."},
	ErrInvalidReason:       {http.StatusBadRequest, "invalid_reason", "The reason must be at most 500 characters."},
	ErrInvalidKey:          {http.StatusBadRequest, "invalid_idempotency_key", "The Idempotency-Key header must be 1 to 100 letters, digits, dashes or underscores."},
	ErrKindMismatch:        {http.StatusBadRequest, "kind_mismatch", "The transaction kind must match the category kind."},
	ErrInvalidLimit:        {http.StatusBadRequest, "invalid_limit", "The budget limit must be a whole number of paise from 0 upwards."},
	ErrInvalidMonth:        {http.StatusBadRequest, "invalid_month", "The month must be written as YYYY-MM."},
	ErrBudgetCommitted:     {http.StatusBadRequest, "budget_committed", "Committed categories are fixed costs and are not budgeted."},
	ErrBudgetNotExpense:    {http.StatusBadRequest, "budget_not_expense", "Only expense categories can have a budget."},
	ErrCommittedIncome:     {http.StatusBadRequest, "invalid_committed", "Only expense categories can be committed."},
	ErrAccountNotFound:     {http.StatusNotFound, "account_not_found", "No account with that ID."},
	ErrCategoryNotFound:    {http.StatusNotFound, "category_not_found", "No category with that ID."},
	ErrTransactionNotFound: {http.StatusNotFound, "transaction_not_found", "No transaction with that ID."},
	ErrDuplicateName:       {http.StatusConflict, "duplicate_name", "That name is already used."},
	ErrAlreadyVoided:       {http.StatusConflict, "already_voided", "That transaction is already void."},
	ErrKeyConflict:         {http.StatusConflict, "idempotency_conflict", "That idempotency key was already used for a different transaction."},
	ErrCategoryInUse:       {http.StatusConflict, "category_in_use", "Cannot delete category referenced by existing transactions; archive it instead."},
}

func (h *Handler) fail(w http.ResponseWriter, action string, err error) {
	for known, problem := range problems {
		if errors.Is(err, known) {
			httpjson.WriteError(w, problem.status, problem.code, problem.message)
			return
		}
	}
	h.logger.Error(action+" failed", "error", err)
	httpjson.WriteError(w, http.StatusInternalServerError, "internal_error", "Something went wrong.")
}

func (h *Handler) badBody(w http.ResponseWriter) {
	httpjson.WriteError(w, http.StatusBadRequest, "invalid_body", "The request body is not valid for this request.")
}

func (h *Handler) pathID(w http.ResponseWriter, r *http.Request, notFound error) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.fail(w, "parse id", notFound)
		return uuid.UUID{}, false
	}
	return id, true
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                string `json:"name"`
		OpeningBalancePaise int64  `json:"openingBalancePaise"`
	}
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.badBody(w)
		return
	}
	account, err := h.service.CreateAccount(r.Context(), req.Name, req.OpeningBalancePaise)
	if err != nil {
		h.fail(w, "create account", err)
		return
	}
	httpjson.Write(w, http.StatusCreated, toAccountResponse(account))
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListAccounts(r.Context())
	if err != nil {
		h.fail(w, "list accounts", err)
		return
	}
	out := make([]accountResponse, 0, len(list))
	for _, a := range list {
		out = append(out, toAccountResponse(a))
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrAccountNotFound)
	if !ok {
		return
	}
	account, err := h.service.GetAccount(r.Context(), id)
	if err != nil {
		h.fail(w, "get account", err)
		return
	}
	httpjson.Write(w, http.StatusOK, toAccountResponse(account))
}

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrAccountNotFound)
	if !ok {
		return
	}
	list, err := h.service.ListTransactions(r.Context(), id)
	if err != nil {
		h.fail(w, "list transactions", err)
		return
	}
	out := make([]transactionResponse, 0, len(list))
	for _, t := range list {
		out = append(out, toTransactionResponse(t))
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Kind      Kind   `json:"kind"`
		Committed bool   `json:"committed"`
	}
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.badBody(w)
		return
	}
	category, err := h.service.CreateCategory(r.Context(), req.Name, req.Kind, req.Committed)
	if err != nil {
		h.fail(w, "create category", err)
		return
	}
	httpjson.Write(w, http.StatusCreated, toCategoryResponse(category))
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.ListCategories(r.Context())
	if err != nil {
		h.fail(w, "list categories", err)
		return
	}
	out := make([]categoryResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toCategoryResponse(c))
	}
	httpjson.Write(w, http.StatusOK, out)
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrCategoryNotFound)
	if !ok {
		return
	}
	category, err := h.service.GetCategory(r.Context(), id)
	if err != nil {
		h.fail(w, "get category", err)
		return
	}
	httpjson.Write(w, http.StatusOK, toCategoryResponse(category))
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrCategoryNotFound)
	if !ok {
		return
	}
	var req struct {
		Name      string `json:"name"`
		Kind      Kind   `json:"kind"`
		Committed bool   `json:"committed"`
		Archived  bool   `json:"archived"`
	}
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.badBody(w)
		return
	}
	category, err := h.service.UpdateCategory(r.Context(), id, req.Name, req.Kind, req.Committed, req.Archived)
	if err != nil {
		h.fail(w, "update category", err)
		return
	}
	httpjson.Write(w, http.StatusOK, toCategoryResponse(category))
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrCategoryNotFound)
	if !ok {
		return
	}
	if err := h.service.DeleteCategory(r.Context(), id); err != nil {
		h.fail(w, "delete category", err)
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) categoryTransactionCount(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrCategoryNotFound)
	if !ok {
		return
	}
	count, err := h.service.CategoryTransactionCount(r.Context(), id)
	if err != nil {
		h.fail(w, "count category transactions", err)
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{"count": count})
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID   string `json:"accountId"`
		CategoryID  string `json:"categoryId"`
		Kind        Kind   `json:"kind"`
		AmountPaise int64  `json:"amountPaise"`
		OccurredOn  string `json:"occurredOn"`
		Note        string `json:"note"`
	}
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.badBody(w)
		return
	}
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		h.fail(w, "create transaction", ErrAccountNotFound)
		return
	}
	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		h.fail(w, "create transaction", ErrCategoryNotFound)
		return
	}
	tx, created, err := h.service.CreateTransaction(r.Context(), NewTransaction{
		AccountID: accountID, CategoryID: categoryID, Kind: req.Kind, AmountPaise: req.AmountPaise,
		OccurredOn: req.OccurredOn, Note: req.Note, IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		h.fail(w, "create transaction", err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	httpjson.Write(w, status, toTransactionResponse(tx))
}

func (h *Handler) voidTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrTransactionNotFound)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength != 0 {
		if err := httpjson.Decode(w, r, &req); err != nil {
			h.badBody(w)
			return
		}
	}
	tx, err := h.service.VoidTransaction(r.Context(), id, req.Reason)
	if err != nil {
		h.fail(w, "void transaction", err)
		return
	}
	httpjson.Write(w, http.StatusOK, toTransactionResponse(tx))
}

type budgetLineResponse struct {
	CategoryID     string `json:"categoryId"`
	CategoryName   string `json:"categoryName"`
	LimitPaise     int64  `json:"limitPaise"`
	SpentPaise     int64  `json:"spentPaise"`
	RemainingPaise int64  `json:"remainingPaise"`
}

type budgetReportResponse struct {
	Month                        string               `json:"month"`
	Lines                        []budgetLineResponse `json:"lines"`
	TotalLimitPaise              int64                `json:"totalLimitPaise"`
	TotalSpentPaise              int64                `json:"totalSpentPaise"`
	TotalRemainingPaise          int64                `json:"totalRemainingPaise"`
	UnbudgetedDiscretionaryPaise int64                `json:"unbudgetedDiscretionaryPaise"`
}

func (h *Handler) setBudget(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrCategoryNotFound)
	if !ok {
		return
	}
	var req struct {
		LimitPaise int64 `json:"limitPaise"`
	}
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.badBody(w)
		return
	}
	budget, err := h.service.SetBudget(r.Context(), id, req.LimitPaise)
	if err != nil {
		h.fail(w, "set budget", err)
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{
		"categoryId": budget.CategoryID.String(),
		"limitPaise": budget.LimitPaise,
		"updatedAt":  budget.UpdatedAt,
	})
}

func (h *Handler) deleteBudget(w http.ResponseWriter, r *http.Request) {
	id, ok := h.pathID(w, r, ErrCategoryNotFound)
	if !ok {
		return
	}
	if err := h.service.DeleteBudget(r.Context(), id); err != nil {
		h.fail(w, "delete budget", err)
		return
	}
	httpjson.Write(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) budgetReport(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.BudgetReport(r.Context(), r.URL.Query().Get("month"))
	if err != nil {
		h.fail(w, "budget report", err)
		return
	}
	lines := make([]budgetLineResponse, 0, len(report.Lines))
	for _, l := range report.Lines {
		lines = append(lines, budgetLineResponse{
			CategoryID: l.CategoryID.String(), CategoryName: l.CategoryName,
			LimitPaise: l.LimitPaise, SpentPaise: l.SpentPaise, RemainingPaise: l.RemainingPaise,
		})
	}
	httpjson.Write(w, http.StatusOK, budgetReportResponse{
		Month: report.Month, Lines: lines, TotalLimitPaise: report.TotalLimitPaise,
		TotalSpentPaise: report.TotalSpentPaise, TotalRemainingPaise: report.TotalRemainingPaise,
		UnbudgetedDiscretionaryPaise: report.UnbudgetedDiscretionaryPaise,
	})
}

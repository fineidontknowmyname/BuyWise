package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
	"selfemi/internal/platform/database"
	"selfemi/internal/platform/httpjson"
)

type purchaseLifecycle struct {
	pool    *pgxpool.Pool
	goals   *goals.Store
	finance *finance.Store
	logger  *slog.Logger
	now     func() time.Time
}

type lifecycleProblem struct {
	code, message string
}

func (e lifecycleProblem) Error() string { return e.message }

var (
	purchaseUnavailable = lifecycleProblem{"goal_inactive", "Only an active or paused goal can be marked purchased."}
	requestChanged      = lifecycleProblem{"purchase_conflict", "This request conflicts with a recorded purchase or allocation. Reload the goal to see the saved result."}
	previewChanged      = lifecycleProblem{"purchase_changed", "The purchase amounts or available funds changed. Review the purchase again."}
	fundsUnavailable    = lifecycleProblem{"funds_unavailable", "There is insufficient eligible cash after reserving funds for other goals and pending surplus."}
	shortfallUnsafe     = lifecycleProblem{"shortfall_unsafe", "Covering the shortfall requires a saved financial profile, current debt, and sufficient emergency savings, cash and monthly operating margin."}
)

type purchaseInput struct {
	Date           string `json:"date"`
	Seller         string `json:"seller"`
	CostPaise      int64  `json:"costPaise"`
	Reference      string `json:"reference"`
	PaymentAccount string `json:"paymentAccountId"`
	CategoryID     string `json:"categoryId"`
}

type namedAmount struct {
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	AmountPaise int64  `json:"amountPaise"`
	TransferID  string `json:"transferId,omitempty"`
}

type purchaseSummary struct {
	purchaseInput
	TargetPaise        int64         `json:"targetPaise"`
	ContributedPaise   int64         `json:"contributedPaise"`
	BackedPaise        int64         `json:"backedPaise"`
	ShortfallPaise     int64         `json:"shortfallPaise"`
	SurplusPaise       int64         `json:"surplusPaise"`
	PaymentAccountName string        `json:"paymentAccountName"`
	CategoryName       string        `json:"categoryName"`
	Funding            []namedAmount `json:"funding"`
	Surplus            []namedAmount `json:"surplus"`
	TransactionID      string        `json:"transactionId,omitempty"`
	RecordedAt         *time.Time    `json:"recordedAt,omitempty"`
}

func (p purchaseLifecycle) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /goals/{id}/purchase", p.route(p.view))
	mux.HandleFunc("POST /goals/{id}/purchase/preview", p.route(p.preview))
	mux.HandleFunc("POST /goals/{id}/purchase", p.route(p.complete))
	mux.HandleFunc("POST /goals/{id}/purchase/allocations", p.route(p.allocate))
}

func (p purchaseLifecycle) route(handler func(http.ResponseWriter, *http.Request, goals.Goal) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			httpjson.WriteError(w, 404, "goal_not_found", "No goal with that ID.")
			return
		}
		if err := database.LockPlanning(ctx, p.pool); err != nil {
			p.fail(w, err)
			return
		}
		goal, err := p.goals.Get(ctx, id)
		if err == nil {
			err = handler(w, r, goal)
		}
		if err != nil {
			p.fail(w, err)
		}
	}
}

func (p purchaseLifecycle) fail(w http.ResponseWriter, err error) {
	var problem lifecycleProblem
	switch {
	case errors.Is(err, goals.ErrNotFound):
		httpjson.WriteError(w, 404, "goal_not_found", "No goal with that ID.")
	case errors.As(err, &problem):
		status := 409
		if strings.HasPrefix(problem.code, "invalid_") {
			status = 400
		}
		httpjson.WriteError(w, status, problem.code, problem.message)
	default:
		p.logger.Error("purchase lifecycle request failed", "error", err)
		httpjson.WriteError(w, 500, "internal_error", "Something went wrong.")
	}
}

func indiaDay(now time.Time) string {
	return now.In(time.FixedZone("IST", 19800)).Format(time.DateOnly)
}

func normalizedID(input string) (string, bool) {
	id, err := uuid.Parse(input)
	return id.String(), err == nil && id != (uuid.UUID{})
}

func (i *purchaseInput) validate(today string) error {
	day, err := time.Parse(time.DateOnly, i.Date)
	account, accountOK := normalizedID(i.PaymentAccount)
	category, categoryOK := normalizedID(i.CategoryID)
	i.Seller, i.Reference = strings.TrimSpace(i.Seller), strings.TrimSpace(i.Reference)
	if err != nil || day.Year() == 0 || i.Date > today || !accountOK || !categoryOK || i.CostPaise < 1 ||
		i.CostPaise > finance.MaxAmountPaise || i.Seller == "" || utf8.RuneCountInString(i.Seller) > 200 ||
		utf8.RuneCountInString(i.Reference) > 500 || strings.ContainsRune(i.Seller+i.Reference, 0) {
		return lifecycleProblem{"invalid_purchase", "Enter a valid purchase date, seller, landed cost, payment account and expense category."}
	}
	i.PaymentAccount, i.CategoryID = account, category
	return nil
}

func (p purchaseLifecycle) plan(ctx context.Context, goal goals.Goal, input purchaseInput, now time.Time) (purchaseSummary, error) {
	out := purchaseSummary{purchaseInput: input, TargetPaise: goal.TargetPricePaise, ContributedPaise: goal.AllocatedPaise,
		Funding: []namedAmount{}, Surplus: []namedAmount{}}
	if goal.Status != "active" && goal.Status != "paused" {
		return out, purchaseUnavailable
	}
	categoryID, _ := uuid.Parse(input.CategoryID)
	category, err := p.finance.GetCategory(ctx, categoryID)
	if errors.Is(err, finance.ErrCategoryNotFound) || err == nil && (category.Kind != finance.Expense || category.Archived) {
		return out, lifecycleProblem{"invalid_category", "Choose an active expense category."}
	}
	if err != nil {
		return out, err
	}
	out.CategoryName = category.Name
	finances, err := p.finance.PurchaseFinances(ctx, now)
	if err != nil {
		return out, err
	}
	reserves, err := p.goals.PurchaseReservations(ctx, goal.ID)
	if err != nil {
		return out, err
	}
	historical, err := p.finance.PurchaseDayBalances(ctx, input.Date)
	if err != nil {
		return out, err
	}
	return out.withFunding(finances, reserves, historical)
}

func (out purchaseSummary) withFunding(finances finance.PurchaseFinances, reserves map[string]goals.AccountReservations, historical map[string]int64) (purchaseSummary, error) {
	var liquid, emergency, paymentFree int64
	for index, account := range finances.Accounts {
		if account.ID == out.PaymentAccount {
			if account.Purpose == "emergency_reserve" {
				break
			}
			out.PaymentAccountName = account.Name
			// Use the payment account's own earmark before drawing from other accounts.
			copy(finances.Accounts[1:index+1], finances.Accounts[:index])
			finances.Accounts[0] = account
			break
		}
	}
	if out.PaymentAccountName == "" {
		return out, lifecycleProblem{"invalid_payment_account", "Choose a payment account outside the emergency reserve."}
	}
	remainingCost := out.CostPaise
	for _, account := range finances.Accounts {
		if account.Purpose == "emergency_reserve" {
			emergency += account.BalancePaise
			continue
		}
		reserve := reserves[account.ID]
		free := account.BalancePaise - reserve.OtherGoals
		pastFree := historical[account.ID] - reserve.OtherGoals
		liquid += free
		backed := max(0, min(reserve.ForGoal, free))
		out.BackedPaise += backed
		spent := max(0, min(remainingCost, backed, pastFree))
		if spent > 0 {
			out.Funding = append(out.Funding, namedAmount{AccountID: account.ID, AccountName: account.Name, AmountPaise: spent})
			remainingCost -= spent
		}
		if surplus := backed - spent; surplus > 0 {
			out.Surplus = append(out.Surplus, namedAmount{AccountID: account.ID, AccountName: account.Name, AmountPaise: surplus})
			out.SurplusPaise += surplus
		}
		if account.ID == out.PaymentAccount {
			paymentFree = min(free-backed, pastFree-spent)
		}
	}
	out.ShortfallPaise = max(0, out.CostPaise-out.BackedPaise)
	if remainingCost != out.ShortfallPaise || paymentFree < out.ShortfallPaise {
		return out, fundsUnavailable
	}
	if out.ShortfallPaise > 0 && (finances.ProfileRevision == 0 || finances.DebtStatus == "" || finances.DebtStatus == "overdue" ||
		emergency < finances.EmergencyTarget || finances.MonthlyMargin < finances.OperatingBuffer ||
		liquid-out.CostPaise < finances.OperatingBuffer) {
		return out, shortfallUnsafe
	}
	return out, nil
}

func previewToken(summary purchaseSummary) (string, error) {
	data, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (p purchaseLifecycle) preview(w http.ResponseWriter, r *http.Request, goal goals.Goal) error {
	var input purchaseInput
	if httpjson.Decode(w, r, &input) != nil {
		return lifecycleProblem{"invalid_purchase", "Enter the purchase details."}
	}
	now := p.now()
	if err := input.validate(indiaDay(now)); err != nil {
		return err
	}
	summary, err := p.plan(r.Context(), goal, input, now)
	if err != nil {
		return err
	}
	token, err := previewToken(summary)
	if err == nil {
		httpjson.Write(w, 200, struct {
			Summary purchaseSummary `json:"summary"`
			Token   string          `json:"token"`
		}{summary, token})
	}
	return err
}

func (p purchaseLifecycle) complete(w http.ResponseWriter, r *http.Request, goal goals.Goal) error {
	var request struct {
		purchaseInput
		Key       string `json:"key"`
		Token     string `json:"token"`
		Confirmed bool   `json:"confirmed"`
	}
	if httpjson.Decode(w, r, &request) != nil {
		return lifecycleProblem{"invalid_purchase", "Enter the purchase details."}
	}
	key, valid := normalizedID(request.Key)
	if !valid || !request.Confirmed {
		return lifecycleProblem{"invalid_confirmation", "Review and confirm the purchase."}
	}
	now := p.now()
	if err := request.purchaseInput.validate(indiaDay(now)); err != nil {
		return err
	}
	ctx := r.Context()
	prior, err := p.goals.Purchase(ctx, goal.ID)
	if err != nil {
		return err
	}
	if prior != nil {
		summary, err := readPurchase(prior)
		if err != nil {
			return err
		}
		if prior.Key != key || summary.purchaseInput != request.purchaseInput {
			return requestChanged
		}
		httpjson.Write(w, 200, summary)
		return nil
	}
	summary, err := p.plan(ctx, goal, request.purchaseInput, now)
	if err != nil {
		return err
	}
	token, err := previewToken(summary)
	if err != nil {
		return err
	}
	if token != request.Token {
		return previewChanged
	}
	for i := range summary.Funding {
		line := &summary.Funding[i]
		if line.AccountID != summary.PaymentAccount {
			line.TransferID, err = p.finance.PostPurchaseTransfer(ctx, line.AccountID, summary.PaymentAccount,
				line.AmountPaise, summary.Date, "Purchase funding: "+goal.Name)
			if err != nil {
				return err
			}
		}
	}
	account, _ := uuid.Parse(summary.PaymentAccount)
	category, _ := uuid.Parse(summary.CategoryID)
	date, _ := time.Parse(time.DateOnly, summary.Date)
	summary.TransactionID, err = p.finance.PostPurchaseExpense(ctx, account, category, summary.CostPaise, date, "Purchase: "+goal.Name, now)
	if err != nil {
		return err
	}
	details, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	lots := make(map[string]int64, len(summary.Surplus))
	for _, line := range summary.Surplus {
		lots[line.AccountID] = line.AmountPaise
	}
	if err := p.goals.CompletePurchase(ctx, goal.ID, key, summary.TransactionID, details, lots); err != nil {
		return err
	}
	prior, err = p.goals.Purchase(ctx, goal.ID)
	if err == nil {
		summary.RecordedAt = &prior.CreatedAt
		httpjson.Write(w, 201, summary)
	}
	return err
}

func readPurchase(event *goals.PurchaseEvent) (*purchaseSummary, error) {
	if event == nil {
		return nil, nil
	}
	var summary purchaseSummary
	err := json.Unmarshal(event.Details, &summary)
	summary.RecordedAt = &event.CreatedAt
	return &summary, err
}

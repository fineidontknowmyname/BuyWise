package finance

import (
	"context"
	"net/http"
	"strings"
	"time"
	"uuid"

	"selfemi/internal/platform/database"
	"selfemi/internal/platform/httpjson"
)

type moneyRule struct {
	CategoryID     string `json:"categoryId"`
	AllowancePaise int64  `json:"allowancePaise"`
	Fixed          bool   `json:"fixed"`
}

type monthlyMoneyPlan struct {
	StartPeriod string      `json:"startPeriod"`
	IncomePaise int64       `json:"incomePaise"`
	Period      string      `json:"period"`
	Rules       []moneyRule `json:"rules"`
}

func (p *monthlyMoneyPlan) startPeriod() string {
	if p.StartPeriod != "" {
		return p.StartPeriod
	}
	return p.Period
}

type moneyCategory struct {
	moneyRule
	Archived       bool   `json:"archived"`
	Name           string `json:"name"`
	MonthlyPaise   int64  `json:"monthlyPaise"`
	SpentPaise     int64  `json:"spentPaise"`
	RemainingPaise int64  `json:"remainingPaise"`
}

type moneyView struct {
	DebtStatus             string          `json:"debtStatus"`
	EmergencyTargetPaise   int64           `json:"emergencyTargetPaise"`
	ReviewRequired         bool            `json:"reviewRequired"`
	PlannedGoalsPaise      int64           `json:"plannedGoalsPaise"`
	UnassignedMonthlyPaise int64           `json:"unassignedMonthlyPaise"`
	BalancePaise           int64           `json:"balancePaise"`
	EmergencyPaise         int64           `json:"emergencyPaise"`
	Revision               int             `json:"revision"`
	Configured             bool            `json:"configured"`
	Period                 string          `json:"period"`
	CurrentPeriod          string          `json:"currentPeriod"`
	IncomePaise            int64           `json:"incomePaise"`
	FixedPaise             int64           `json:"fixedPaise"`
	FlexiblePaise          int64           `json:"flexiblePaise"`
	MonthlySavingsPaise    int64           `json:"monthlySavingsPaise"`
	MonthlyShortfallPaise  int64           `json:"monthlyShortfallPaise"`
	ProtectedPaise         int64           `json:"protectedPaise"`
	ReservedPaise          int64           `json:"reservedPaise"`
	SavingsNowPaise        int64           `json:"savingsNowPaise"`
	CashShortfallPaise     int64           `json:"cashShortfallPaise"`
	Categories             []moneyCategory `json:"categories"`
}

func monthKey(now time.Time) string { return now.In(businessZone).Format("2006-01") }

func (s *Store) moneyCapacity(ctx context.Context, plan *monthlyMoneyPlan) (capacity, error) {
	budgets, err := s.ListBudgets(ctx)
	if err != nil {
		return capacity{}, err
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return capacity{}, err
	}
	result := capacity{Version: "savings-v3", FixedIncomePaise: plan.IncomePaise}
	for _, category := range categories {
		if category.Kind != Expense || category.Archived {
			continue
		}
		for _, budget := range budgets {
			if budget.CategoryID != category.ID {
				continue
			}
			if category.Committed {
				result.CommittedPaise += budget.LimitPaise
			} else {
				result.DiscretionaryPaise += budget.LimitPaise
			}
		}
	}
	result.CategoryBudgetPaise = result.CommittedPaise + result.DiscretionaryPaise
	result.SafeSavingsPaise = max(0, plan.IncomePaise-result.CategoryBudgetPaise)
	return result, nil
}

func (s *Store) moneyOverview(ctx context.Context, profile financialProfile, now time.Time) (moneyView, error) {
	v := moneyView{Revision: profile.Revision, Configured: profile.MoneyPlan != nil, CurrentPeriod: monthKey(now), Period: monthKey(now), Categories: []moneyCategory{}}
	v.DebtStatus, v.EmergencyTargetPaise = profile.DebtStatus, profile.EmergencyTargetPaise
	plan := profile.MoneyPlan
	if plan == nil {
		plan = &monthlyMoneyPlan{Period: v.Period, IncomePaise: profile.estimate().FixedIncomePaise}
	} else {
		v.Period = plan.Period
	}
	v.ReviewRequired = v.Configured && v.Period < v.CurrentPeriod
	estimate, err := s.moneyCapacity(ctx, plan)
	if err != nil {
		return v, err
	}
	v.IncomePaise, v.FixedPaise, v.FlexiblePaise, v.MonthlySavingsPaise = estimate.FixedIncomePaise, estimate.CommittedPaise, estimate.DiscretionaryPaise, estimate.SafeSavingsPaise
	v.MonthlyShortfallPaise = max(0, v.FixedPaise+v.FlexiblePaise-v.IncomePaise)
	if s.commitments != nil {
		v.PlannedGoalsPaise, err = s.commitments(ctx)
		if err != nil {
			return v, err
		}
	}
	v.UnassignedMonthlyPaise = max(0, v.MonthlySavingsPaise-v.PlannedGoalsPaise)
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return v, err
	}
	budgets, err := s.ListBudgets(ctx)
	if err != nil {
		return v, err
	}
	start, err := time.Parse("2006-01", plan.startPeriod())
	if err != nil {
		return v, err
	}
	end := now.In(businessZone)
	end = time.Date(end.Year(), end.Month(), end.Day()+1, 0, 0, 0, 0, businessZone)
	spent, err := s.SpentByCategory(ctx, start, end)
	if err != nil {
		return v, err
	}
	periodStart, err := time.Parse("2006-01", v.Period)
	if err != nil {
		return v, err
	}
	currentSpent, err := s.SpentByCategory(ctx, periodStart, end)
	if err != nil {
		return v, err
	}
	seen := map[string]bool{}
	for _, category := range categories {
		id := category.ID.String()
		var monthly int64
		budgeted := false
		for _, budget := range budgets {
			if budget.CategoryID == category.ID {
				monthly = budget.LimitPaise
				budgeted = true
			}
		}
		if !v.Configured && !budgeted {
			for _, item := range profile.Items {
				if item.Kind != "fixed_expense" && item.Kind != "essential_expense" && item.Kind != "debt" && item.Kind != "discretionary" {
					continue
				}
				if profile.BudgetCategoryLinks[item.Name] != id && !strings.EqualFold(strings.TrimSpace(item.Name), category.Name) {
					continue
				}
				amount := item.AmountPaise
				switch item.Frequency {
				case "weekly":
					amount = (amount*52 + 11) / 12
				case "biweekly":
					amount = (amount*26 + 11) / 12
				case "annual":
					amount = (amount + 11) / 12
				case "monthly":
				default:
					continue
				}
				monthly += amount
			}
		}
		rule := moneyRule{CategoryID: id, AllowancePaise: monthly + spent[category.ID] - currentSpent[category.ID], Fixed: category.Committed}
		found := false
		for _, saved := range plan.Rules {
			if saved.CategoryID == id {
				rule = saved
				found = true
			}
		}
		if category.Kind != Expense || (category.Archived && !found) {
			continue
		}
		if category.Archived {
			monthly = 0
		}
		rule.Fixed = rule.Fixed || category.Committed
		remaining := max(0, rule.AllowancePaise-spent[category.ID])
		v.Categories = append(v.Categories, moneyCategory{moneyRule: rule, Archived: category.Archived, Name: category.Name, MonthlyPaise: monthly, SpentPaise: currentSpent[category.ID], RemainingPaise: remaining})
		if v.Configured {
			v.ProtectedPaise += remaining
		}
		seen[id] = true
	}
	for _, rule := range plan.Rules {
		if !seen[rule.CategoryID] {
			id, _ := uuid.Parse(rule.CategoryID)
			v.ProtectedPaise += max(0, rule.AllowancePaise-spent[id])
		}
	}
	day := now.In(businessZone).Format(DateLayout)
	accounts, err := s.accountSummaries(ctx, &day)
	if err != nil {
		return v, err
	}
	purposes, err := s.designations(ctx)
	if err != nil {
		return v, err
	}
	reserved := map[string]int64{}
	if s.reservations != nil {
		reserved, err = s.reservations(ctx)
		if err != nil {
			return v, err
		}
	}
	var cash int64
	for _, account := range accounts {
		v.BalancePaise += account.BalancePaise
		if purposes[account.ID.String()] == "emergency_reserve" {
			v.EmergencyPaise += max(0, account.BalancePaise)
			continue
		}
		cash += account.BalancePaise
		v.ReservedPaise += reserved[account.ID.String()]
	}
	v.SavingsNowPaise = max(0, cash-v.ProtectedPaise-v.ReservedPaise)
	if v.ReviewRequired {
		v.SavingsNowPaise = 0
	}
	v.CashShortfallPaise = max(0, v.ProtectedPaise+v.ReservedPaise-cash)
	return v, nil
}

func (h foundationHandler) getMoneyPlan(w http.ResponseWriter, r *http.Request) {
	profile, err := h.store.loadProfile(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	view, err := h.store.moneyOverview(r.Context(), profile, h.today())
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, 200, view)
}

func (h foundationHandler) putMoneyPlan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DebtStatus           *string `json:"debtStatus"`
		EmergencyTargetPaise *int64  `json:"emergencyTargetPaise"`
		Revision             int     `json:"revision"`
		IncomePaise          int64   `json:"incomePaise"`
		Categories           []struct {
			CategoryID  string `json:"categoryId"`
			AmountPaise int64  `json:"amountPaise"`
		} `json:"categories"`
	}
	if httpjson.Decode(w, r, &input) != nil || input.IncomePaise < 0 || input.IncomePaise > MaxAmountPaise || len(input.Categories) > 100 {
		httpjson.WriteError(w, 400, "invalid_money_plan", "Enter a monthly income and valid category amounts.")
		return
	}
	ctx := r.Context()
	if input.DebtStatus != nil && *input.DebtStatus != "" && *input.DebtStatus != "current" && *input.DebtStatus != "overdue" {
		httpjson.WriteError(w, 400, "invalid_debt_status", "Choose a valid repayment status.")
		return
	}
	if input.EmergencyTargetPaise != nil && (*input.EmergencyTargetPaise < 0 || *input.EmergencyTargetPaise > MaxAmountPaise) {
		httpjson.WriteError(w, 400, "invalid_reserve_target", "Enter a valid emergency reserve target.")
		return
	}
	if err := database.LockPlanning(ctx, h.store.pool); err != nil {
		h.fail(w, err)
		return
	}
	profile, err := h.store.loadProfile(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	if input.Revision != profile.Revision {
		h.fail(w, errProfileConflict)
		return
	}
	categories, err := h.store.ListCategories(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	valid := map[string]Category{}
	for _, category := range categories {
		if category.Kind == Expense && !category.Archived {
			valid[category.ID.String()] = category
		}
	}
	seen := map[string]bool{}
	for _, line := range input.Categories {
		if _, ok := valid[line.CategoryID]; !ok || seen[line.CategoryID] || line.AmountPaise < 0 || line.AmountPaise > MaxAmountPaise {
			httpjson.WriteError(w, 400, "invalid_category", "Choose your active expense categories and valid amounts.")
			return
		}
		seen[line.CategoryID] = true
	}
	if profile.MoneyPlan == nil {
		profile.MoneyPlan = &monthlyMoneyPlan{Period: monthKey(h.today()), Rules: []moneyRule{}}
	}
	if profile.MoneyPlan.StartPeriod == "" {
		profile.MoneyPlan.StartPeriod = profile.MoneyPlan.Period
	}
	start, _ := time.Parse("2006-01", profile.MoneyPlan.startPeriod())
	period, _ := time.Parse("2006-01", profile.MoneyPlan.Period)
	priorSpent, err := h.store.SpentByCategory(ctx, start, period)
	if err != nil {
		h.fail(w, err)
		return
	}
	profile.MoneyPlan.IncomePaise = input.IncomePaise
	if input.DebtStatus != nil {
		profile.DebtStatus = *input.DebtStatus
	}
	if input.EmergencyTargetPaise != nil {
		profile.EmergencyTargetPaise = *input.EmergencyTargetPaise
	}
	budgets, err := h.store.ListBudgets(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	previous := map[string]int64{}
	for _, budget := range budgets {
		previous[budget.CategoryID.String()] = budget.LimitPaise
	}
	for _, line := range input.Categories {
		category := valid[line.CategoryID]
		if err := h.store.UpsertBudget(ctx, Budget{CategoryID: category.ID, LimitPaise: line.AmountPaise, UpdatedAt: h.today()}); err != nil {
			h.fail(w, err)
			return
		}
		found := false
		for i := range profile.MoneyPlan.Rules {
			rule := &profile.MoneyPlan.Rules[i]
			if rule.CategoryID == line.CategoryID {
				rule.Fixed = rule.Fixed || category.Committed
				rule.AllowancePaise += max(0, line.AmountPaise-previous[line.CategoryID])
				found = true
			}
		}
		if !found {
			profile.MoneyPlan.Rules = append(profile.MoneyPlan.Rules, moneyRule{line.CategoryID, line.AmountPaise + priorSpent[category.ID], category.Committed})
		}
	}
	profile, err = h.store.saveProfile(ctx, profile)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpjson.Write(w, 200, profile)
}

func (h foundationHandler) moveMoney(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Revision    int    `json:"revision"`
		CategoryID  string `json:"categoryId"`
		AmountPaise int64  `json:"amountPaise"`
	}
	if httpjson.Decode(w, r, &input) != nil || input.AmountPaise <= 0 || input.AmountPaise > MaxAmountPaise {
		httpjson.WriteError(w, 400, "invalid_move", "Enter a positive amount to move to savings.")
		return
	}
	if err := database.LockPlanning(r.Context(), h.store.pool); err != nil {
		h.fail(w, err)
		return
	}
	profile, err := h.store.loadProfile(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	if input.Revision != profile.Revision {
		h.fail(w, errProfileConflict)
		return
	}
	view, err := h.store.moneyOverview(r.Context(), profile, h.today())
	if err != nil {
		h.fail(w, err)
		return
	}
	if profile.MoneyPlan != nil {
		for _, category := range view.Categories {
			if category.CategoryID != input.CategoryID || category.Fixed || input.AmountPaise > max(0, category.RemainingPaise-view.CashShortfallPaise) {
				continue
			}
			for i := range profile.MoneyPlan.Rules {
				rule := &profile.MoneyPlan.Rules[i]
				if rule.CategoryID != input.CategoryID {
					continue
				}
				rule.AllowancePaise -= input.AmountPaise
				if _, err := h.store.saveProfile(r.Context(), profile); err != nil {
					h.fail(w, err)
					return
				}
				w.WriteHeader(204)
				return
			}
		}
	}
	httpjson.WriteError(w, 400, "protected_money", "Only funded, unused flexible spending can move to savings. Fixed obligations stay protected.")
}

func (h foundationHandler) nextMoneyMonth(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Revision       int    `json:"revision"`
		FlexibleChoice string `json:"flexibleChoice"`
	}
	if httpjson.Decode(w, r, &input) != nil || (input.FlexibleChoice != "carry" && input.FlexibleChoice != "save") {
		httpjson.WriteError(w, 400, "review_required", "Choose whether unused flexible money carries forward or becomes savings.")
		return
	}
	ctx := r.Context()
	if err := database.LockPlanning(ctx, h.store.pool); err != nil {
		h.fail(w, err)
		return
	}
	profile, err := h.store.loadProfile(ctx)
	if err != nil {
		h.fail(w, err)
		return
	}
	if input.Revision != profile.Revision {
		h.fail(w, errProfileConflict)
		return
	}
	if profile.MoneyPlan == nil || profile.MoneyPlan.Period >= monthKey(h.today()) {
		httpjson.WriteError(w, 400, "month_not_due", "The current month is already open.")
		return
	}
	if profile.MoneyPlan.StartPeriod == "" {
		profile.MoneyPlan.StartPeriod = profile.MoneyPlan.Period
	}
	start, _ := time.Parse("2006-01", profile.MoneyPlan.startPeriod())
	current, _ := time.Parse("2006-01", monthKey(h.today()))
	spent, err := h.store.SpentByCategory(ctx, start, current)
	if err != nil {
		h.fail(w, err)
		return
	}
	view, err := h.store.moneyOverview(ctx, profile, h.today())
	if err != nil {
		h.fail(w, err)
		return
	}
	rules := []moneyRule{}
	for _, category := range view.Categories {
		id, _ := uuid.Parse(category.CategoryID)
		allowance := max(category.AllowancePaise, spent[id])
		if !category.Fixed && input.FlexibleChoice == "save" {
			allowance = spent[id]
		}
		rules = append(rules, moneyRule{category.CategoryID, category.MonthlyPaise + allowance, category.Fixed})
	}
	profile.MoneyPlan.Rules = rules
	profile.MoneyPlan.Period = monthKey(h.today())
	if _, err := h.store.saveProfile(ctx, profile); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(204)
}

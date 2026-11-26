package finance

import (
	"context"
	"time"

	"selfemi/internal/platform/database"
)

func (p financialProfile) estimateWithBudgets(budgets []Budget) capacity {
	if !p.UseCategoryBudgets {
		return p.estimate()
	}
	if len(budgets) == 0 {
		result := p.estimate()
		result.Version = "monthly-v2-budget-fallback"
		return result
	}
	linked := make(map[string]bool, len(budgets))
	var total int64
	for _, budget := range budgets {
		linked[budget.CategoryID.String()] = true
		total += budget.LimitPaise
	}
	copy := p
	copy.Items = make([]profileItem, 0, len(p.Items)+1)
	for _, item := range p.Items {
		income := item.Kind == "fixed_income" || item.Kind == "variable_income" || item.Kind == "irregular_income"
		if item.Kind == "discretionary" || (!income && linked[p.BudgetCategoryLinks[item.Name]]) {
			continue
		}
		copy.Items = append(copy.Items, item)
	}
	copy.Items = append(copy.Items, profileItem{Name: "Linked category budgets", Kind: "discretionary", Frequency: "monthly", AmountPaise: total})
	result := copy.estimate()
	result.Version = "monthly-v2-budgets"
	result.CategoryBudgetPaise = total
	return result
}

func (s *Store) profileCapacity(ctx context.Context, profile financialProfile) (capacity, error) {
	if !profile.UseCategoryBudgets {
		return profile.estimate(), nil
	}
	budgets, err := s.ListBudgets(ctx)
	if err != nil {
		return capacity{}, err
	}
	return profile.estimateWithBudgets(budgets), nil
}

type setupBudget struct {
	CategoryID  string `json:"categoryId"`
	Name        string `json:"name"`
	AmountPaise int64  `json:"amountPaise"`
}

type setupSuggestion struct {
	CategoryID     string `json:"categoryId"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	AmountPaise    int64  `json:"amountPaise"`
	MonthsObserved int    `json:"monthsObserved"`
}

type setupConnections struct {
	Budgets        []setupBudget     `json:"budgets"`
	Suggestions    []setupSuggestion `json:"suggestions"`
	HistoryFrom    string            `json:"historyFrom"`
	HistoryThrough string            `json:"historyThrough"`
}

func (s *Store) setupConnections(ctx context.Context, now time.Time) (setupConnections, error) {
	end := now.In(businessZone)
	end = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, businessZone)
	start := end.AddDate(0, -3, 0)
	result := setupConnections{Budgets: []setupBudget{}, Suggestions: []setupSuggestion{}, HistoryFrom: start.Format(DateLayout), HistoryThrough: end.AddDate(0, 0, -1).Format(DateLayout)}
	budgets, err := s.ListBudgets(ctx)
	if err != nil {
		return result, err
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return result, err
	}
	budgeted := make(map[string]bool)
	for _, budget := range budgets {
		for _, category := range categories {
			if category.ID == budget.CategoryID {
				result.Budgets = append(result.Budgets, setupBudget{category.ID.String(), category.Name, budget.LimitPaise})
				budgeted[category.ID.String()] = true
				break
			}
		}
	}
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT c.id::text, c.name, t.kind,
		SUM(t.amount_paise)::bigint, COUNT(DISTINCT date_trunc('month', t.occurred_on))
		FROM finance.transactions t JOIN finance.categories c ON c.id = t.category_id
		WHERE t.voided_at IS NULL AND NOT c.archived AND t.occurred_on >= $1::date AND t.occurred_on < $2::date
		GROUP BY c.id, c.name, t.kind ORDER BY c.name LIMIT 100`, start.Format(DateLayout), end.Format(DateLayout))
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item setupSuggestion
		var total int64
		if err := rows.Scan(&item.CategoryID, &item.Name, &item.Kind, &total, &item.MonthsObserved); err != nil {
			return result, err
		}
		if item.Kind == "income" {
			if item.MonthsObserved < 2 {
				continue
			}
			item.Kind = "variable_income"
			item.AmountPaise = total / 3
		} else {
			if budgeted[item.CategoryID] {
				continue
			}
			item.Kind = "essential_expense"
			item.AmountPaise = (total + 2) / 3
		}
		if item.AmountPaise > 0 && item.AmountPaise <= MaxAmountPaise {
			result.Suggestions = append(result.Suggestions, item)
		}
	}
	return result, rows.Err()
}

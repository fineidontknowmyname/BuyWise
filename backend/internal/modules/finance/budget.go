package finance

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"uuid"
)

const MonthLayout = "2006-01"

var (
	ErrInvalidLimit     = errors.New("limit must be a whole number of paise from 0 to 1,000,000,000,000")
	ErrInvalidMonth     = errors.New("month must be YYYY-MM")
	ErrBudgetCommitted  = errors.New("committed categories are fixed costs and are not budgeted")
	ErrBudgetNotExpense = errors.New("only expense categories can have a budget")
)

type Budget struct {
	CategoryID uuid.UUID
	LimitPaise int64
	UpdatedAt  time.Time
}

type BudgetLine struct {
	CategoryID     uuid.UUID
	CategoryName   string
	LimitPaise     int64
	SpentPaise     int64
	RemainingPaise int64
}

type BudgetReport struct {
	Month                        string
	Lines                        []BudgetLine
	TotalLimitPaise              int64
	TotalSpentPaise              int64
	TotalRemainingPaise          int64
	UnbudgetedDiscretionaryPaise int64
}

type BudgetRepository interface {
	UpsertBudget(ctx context.Context, budget Budget) error
	ListBudgets(ctx context.Context) ([]Budget, error)
	SpentByCategory(ctx context.Context, fromInclusive, toExclusive time.Time) (map[uuid.UUID]int64, error)
}

func (s *Service) SetBudget(ctx context.Context, categoryID uuid.UUID, limitPaise int64) (Budget, error) {
	if limitPaise < 0 || limitPaise > MaxAmountPaise {
		return Budget{}, ErrInvalidLimit
	}
	category, err := s.repo.GetCategory(ctx, categoryID)
	if err != nil {
		return Budget{}, err
	}
	if category.Kind != Expense {
		return Budget{}, ErrBudgetNotExpense
	}
	if category.Committed {
		return Budget{}, ErrBudgetCommitted
	}
	budget := Budget{CategoryID: categoryID, LimitPaise: limitPaise, UpdatedAt: s.now().UTC()}
	if err := s.repo.UpsertBudget(ctx, budget); err != nil {
		return Budget{}, err
	}
	return budget, nil
}

func (s *Service) BudgetReport(ctx context.Context, month string) (BudgetReport, error) {
	first := s.Today()
	first = time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, time.UTC)
	if month != "" {
		parsed, err := time.Parse(MonthLayout, month)
		if err != nil {
			return BudgetReport{}, ErrInvalidMonth
		}
		first = parsed
	}
	next := first.AddDate(0, 1, 0)

	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		return BudgetReport{}, err
	}
	budgets, err := s.repo.ListBudgets(ctx)
	if err != nil {
		return BudgetReport{}, err
	}
	spent, err := s.repo.SpentByCategory(ctx, first, next)
	if err != nil {
		return BudgetReport{}, err
	}

	limits := make(map[uuid.UUID]int64, len(budgets))
	for _, b := range budgets {
		limits[b.CategoryID] = b.LimitPaise
	}

	report := BudgetReport{Month: first.Format(MonthLayout), Lines: []BudgetLine{}}
	for _, c := range categories {
		if c.Kind != Expense || c.Committed {
			continue
		}
		limit, budgeted := limits[c.ID]
		if !budgeted {
			report.UnbudgetedDiscretionaryPaise += spent[c.ID]
			continue
		}
		line := BudgetLine{
			CategoryID: c.ID, CategoryName: c.Name, LimitPaise: limit,
			SpentPaise: spent[c.ID], RemainingPaise: limit - spent[c.ID],
		}
		report.Lines = append(report.Lines, line)
		report.TotalLimitPaise += line.LimitPaise
		report.TotalSpentPaise += line.SpentPaise
		report.TotalRemainingPaise += line.RemainingPaise
	}
	sort.SliceStable(report.Lines, func(i, j int) bool {
		return strings.ToLower(report.Lines[i].CategoryName) < strings.ToLower(report.Lines[j].CategoryName)
	})
	return report, nil
}

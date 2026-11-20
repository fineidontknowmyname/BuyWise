package finance

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"selfemi/internal/platform/migrate"
)

func (f fixture) discretionary(t *testing.T, name string) Category {
	t.Helper()
	c, err := f.svc.CreateCategory(context.Background(), name, Expense, false)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f fixture) spend(t *testing.T, category Category, amount int64, date, key string) Transaction {
	t.Helper()
	tx, _, err := f.svc.CreateTransaction(context.Background(), NewTransaction{
		AccountID: f.account.ID, CategoryID: category.ID, Kind: Expense,
		AmountPaise: amount, OccurredOn: date, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestSetBudgetRules(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	food := f.discretionary(t, "Groceries")

	if _, err := f.svc.SetBudget(ctx, food.ID, 8_000_00); err != nil {
		t.Fatalf("valid budget: %v", err)
	}
	if _, err := f.svc.SetBudget(ctx, food.ID, 0); err != nil {
		t.Errorf("zero budget should be allowed: %v", err)
	}
	for _, bad := range []int64{-1, MaxAmountPaise + 1} {
		if _, err := f.svc.SetBudget(ctx, food.ID, bad); !errors.Is(err, ErrInvalidLimit) {
			t.Errorf("limit %d err = %v", bad, err)
		}
	}
	if _, err := f.svc.SetBudget(ctx, f.expense.ID, 100); !errors.Is(err, ErrBudgetCommitted) {
		t.Errorf("committed err = %v", err)
	}
	if _, err := f.svc.SetBudget(ctx, f.income.ID, 100); !errors.Is(err, ErrBudgetNotExpense) {
		t.Errorf("income err = %v", err)
	}
	if _, err := f.svc.SetBudget(ctx, uuid.NewV7(), 100); !errors.Is(err, ErrCategoryNotFound) {
		t.Errorf("unknown err = %v", err)
	}
}

func TestSetBudgetReplacesTheLimit(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	food := f.discretionary(t, "Groceries")
	f.svc.SetBudget(ctx, food.ID, 5_000)
	f.svc.SetBudget(ctx, food.ID, 7_000)
	report, err := f.svc.BudgetReport(ctx, "")
	if err != nil || len(report.Lines) != 1 || report.Lines[0].LimitPaise != 7_000 {
		t.Fatalf("report = %+v err %v", report, err)
	}
}

func TestReportSpentRemainingAndTotals(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	food, fun := f.discretionary(t, "Groceries"), f.discretionary(t, "Fun")
	f.svc.SetBudget(ctx, food.ID, 10_000)
	f.svc.SetBudget(ctx, fun.ID, 2_000)
	f.spend(t, food, 3_000, "2026-10-02", "a")
	f.spend(t, food, 1_500, "2026-10-20", "b")
	f.spend(t, fun, 2_500, "2026-10-03", "c")

	report, err := f.svc.BudgetReport(ctx, "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	if report.Month != "2026-10" || len(report.Lines) != 2 {
		t.Fatalf("report = %+v", report)
	}
	if report.Lines[0].CategoryName != "Fun" || report.Lines[1].CategoryName != "Groceries" {
		t.Fatalf("lines should be sorted by name: %+v", report.Lines)
	}
	fun0, food0 := report.Lines[0], report.Lines[1]
	if fun0.SpentPaise != 2_500 || fun0.RemainingPaise != -500 {
		t.Errorf("overspent line = %+v, want remaining -500", fun0)
	}
	if food0.SpentPaise != 4_500 || food0.RemainingPaise != 5_500 {
		t.Errorf("food line = %+v", food0)
	}
	if report.TotalLimitPaise != 12_000 || report.TotalSpentPaise != 7_000 || report.TotalRemainingPaise != 5_000 {
		t.Errorf("totals = %+v", report)
	}
}

func TestReportIgnoresVoidIncomeCommittedAndOtherMonths(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	food := f.discretionary(t, "Groceries")
	f.svc.SetBudget(ctx, food.ID, 10_000)
	f.spend(t, food, 1_000, "2026-10-05", "keep")
	voided := f.spend(t, food, 9_000, "2026-10-06", "void")
	if _, err := f.svc.VoidTransaction(ctx, voided.ID, ""); err != nil {
		t.Fatal(err)
	}
	f.spend(t, food, 400, "2026-09-30", "september")
	f.spend(t, food, 700, "2026-11-01", "november")
	f.spend(t, f.expense, 50_000, "2026-10-01", "rent")
	f.svc.CreateTransaction(ctx, f.tx(Income, 99_000, "salary"))

	report, _ := f.svc.BudgetReport(ctx, "2026-10")
	if report.TotalSpentPaise != 1_000 || report.UnbudgetedDiscretionaryPaise != 0 {
		t.Fatalf("report = %+v, want only the 1,000 spent on groceries in October", report)
	}
	other, _ := f.svc.BudgetReport(ctx, "2026-09")
	if other.TotalSpentPaise != 400 {
		t.Errorf("september spent = %d", other.TotalSpentPaise)
	}
}

func TestReportMonthBoundaries(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	food := f.discretionary(t, "Groceries")
	f.svc.SetBudget(ctx, food.ID, 10_000)
	f.spend(t, food, 1, "2026-10-01", "first")
	f.spend(t, food, 2, "2026-10-31", "last")
	f.spend(t, food, 4, "2026-11-01", "next")
	f.spend(t, food, 8, "2026-09-30", "prev")
	report, _ := f.svc.BudgetReport(ctx, "2026-10")
	if report.TotalSpentPaise != 3 {
		t.Fatalf("spent = %d, want first and last day of October only", report.TotalSpentPaise)
	}
	feb, _ := f.svc.BudgetReport(ctx, "2028-02")
	if feb.Month != "2028-02" {
		t.Fatalf("month = %s", feb.Month)
	}
	f.spend(t, food, 16, "2028-02-29", "leap")
	feb, _ = f.svc.BudgetReport(ctx, "2028-02")
	if feb.TotalSpentPaise != 16 {
		t.Fatalf("leap day spent = %d", feb.TotalSpentPaise)
	}
}

func TestReportDefaultsToTheCurrentMonthInIndia(t *testing.T) {
	late := NewService(newMemoryRepo(), func() time.Time { return time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC) })
	report, err := late.BudgetReport(context.Background(), "")
	if err != nil || report.Month != "2026-11" {
		t.Fatalf("19:00 UTC on Oct 31 is already November in India; got %+v err %v", report, err)
	}
	early := NewService(newMemoryRepo(), func() time.Time { return time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC) })
	if report, _ := early.BudgetReport(context.Background(), ""); report.Month != "2026-10" {
		t.Fatalf("month = %s", report.Month)
	}
}

func TestReportShowsUnbudgetedDiscretionarySpending(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	food, fun := f.discretionary(t, "Groceries"), f.discretionary(t, "Fun")
	f.svc.SetBudget(ctx, food.ID, 10_000)
	f.spend(t, fun, 750, "2026-10-04", "fun")
	report, _ := f.svc.BudgetReport(ctx, "2026-10")
	if report.UnbudgetedDiscretionaryPaise != 750 || report.TotalSpentPaise != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestReportInvalidMonthAndEmpty(t *testing.T) {
	f := newFixture(t)
	for _, bad := range []string{"2026-13", "2026-1", "10-2026", "2026-10-01", "oct"} {
		if _, err := f.svc.BudgetReport(context.Background(), bad); !errors.Is(err, ErrInvalidMonth) {
			t.Errorf("month %q err = %v", bad, err)
		}
	}
	report, err := f.svc.BudgetReport(context.Background(), "2026-10")
	if err != nil || report.Lines == nil || len(report.Lines) != 0 || report.TotalLimitPaise != 0 {
		t.Fatalf("empty report = %+v err %v", report, err)
	}
}

func TestBudgetHTTP(t *testing.T) {
	h := newHTTP(t)
	_, cat := call(t, h, "POST", "/categories", `{"name":"Fun","kind":"expense"}`, "")
	_, rent := call(t, h, "POST", "/categories", `{"name":"Rent","kind":"expense","committed":true}`, "")
	id, rentID := cat["id"].(string), rent["id"].(string)

	code, out := call(t, h, "PUT", "/budgets/"+id, `{"limitPaise":250000}`, "")
	if code != 200 || out["limitPaise"] != float64(250000) {
		t.Fatalf("set: %d %v", code, out)
	}
	code, report := call(t, h, "GET", "/budgets?month=2026-10", "", "")
	lines, _ := report["lines"].([]any)
	if code != 200 || report["month"] != "2026-10" || len(lines) != 1 || report["totalLimitPaise"] != float64(250000) {
		t.Fatalf("report: %d %v", code, report)
	}
	cases := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"committed", "PUT", "/budgets/" + rentID, `{"limitPaise":1}`, 400, "budget_committed"},
		{"negative", "PUT", "/budgets/" + id, `{"limitPaise":-1}`, 400, "invalid_limit"},
		{"fraction", "PUT", "/budgets/" + id, `{"limitPaise":1.5}`, 400, "invalid_body"},
		{"unknown category", "PUT", "/budgets/not-a-uuid", `{"limitPaise":1}`, 404, "category_not_found"},
		{"bad month", "GET", "/budgets?month=2026-13", "", 400, "invalid_month"},
	}
	for _, c := range cases {
		status, out := call(t, h, c.method, c.path, c.body, "")
		if status != c.status || errorCode(out) != c.code {
			t.Errorf("%s: got %d %q, want %d %q", c.name, status, errorCode(out), c.status, c.code)
		}
	}
}

func TestStoreBudgetsAndSpending(t *testing.T) {
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST is not set; skipping database test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := migrate.Apply(ctx, pool, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE finance.budgets, finance.transactions, finance.accounts CASCADE"); err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewStore(pool), func() time.Time { return clockUTC })

	groceries := categoryNamed(t, svc, "Groceries")
	shopping := categoryNamed(t, svc, "Shopping")
	rent := categoryNamed(t, svc, "Rent")
	account, _ := svc.CreateAccount(ctx, "Main", 0)
	add := func(c Category, amount int64, date, key string) Transaction {
		tx, _, err := svc.CreateTransaction(ctx, NewTransaction{AccountID: account.ID, CategoryID: c.ID, Kind: Expense, AmountPaise: amount, OccurredOn: date, IdempotencyKey: key})
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
	add(groceries, 3_000, "2026-10-01", "a")
	add(groceries, 2_000, "2026-10-31", "b")
	add(groceries, 9_999, "2026-11-01", "c")
	add(groceries, 8_888, "2026-09-30", "d")
	add(rent, 50_000, "2026-10-01", "e")
	add(shopping, 700, "2026-10-10", "f")
	voided := add(groceries, 4_444, "2026-10-12", "g")
	svc.VoidTransaction(ctx, voided.ID, "")

	if _, err := svc.SetBudget(ctx, groceries.ID, 10_000); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetBudget(ctx, groceries.ID, 12_000); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetBudget(ctx, rent.ID, 1); !errors.Is(err, ErrBudgetCommitted) {
		t.Fatalf("committed err = %v", err)
	}

	report, err := svc.BudgetReport(ctx, "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Lines) != 1 {
		t.Fatalf("lines = %+v", report.Lines)
	}
	line := report.Lines[0]
	if line.LimitPaise != 12_000 || line.SpentPaise != 5_000 || line.RemainingPaise != 7_000 {
		t.Fatalf("line = %+v", line)
	}
	if report.UnbudgetedDiscretionaryPaise != 700 {
		t.Fatalf("unbudgeted = %d", report.UnbudgetedDiscretionaryPaise)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO finance.budgets (category_id, limit_paise, updated_at) VALUES ($1::uuid, -1, now())`, shopping.ID.String()); err == nil {
		t.Fatal("database should reject a negative limit")
	}
}

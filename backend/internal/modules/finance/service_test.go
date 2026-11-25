package finance

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"
)

var clockUTC = time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC)

type fixture struct {
	svc     *Service
	account AccountSummary
	income  Category
	expense Category
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	svc := NewService(newMemoryRepo(), func() time.Time { return clockUTC })
	ctx := context.Background()
	account, err := svc.CreateAccount(ctx, "Savings", 100_000)
	if err != nil {
		t.Fatal(err)
	}
	income, err := svc.CreateCategory(ctx, "Salary", Income, false)
	if err != nil {
		t.Fatal(err)
	}
	expense, err := svc.CreateCategory(ctx, "Rent", Expense, true)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{svc: svc, account: account, income: income, expense: expense}
}

func (f fixture) tx(kind Kind, amount int64, key string) NewTransaction {
	category := f.expense
	if kind == Income {
		category = f.income
	}
	return NewTransaction{
		AccountID: f.account.ID, CategoryID: category.ID, Kind: kind,
		AmountPaise: amount, OccurredOn: "2026-10-01", IdempotencyKey: key,
	}
}

func (f fixture) balance(t *testing.T) int64 {
	t.Helper()
	a, err := f.svc.GetAccount(context.Background(), f.account.ID)
	if err != nil {
		t.Fatal(err)
	}
	return a.BalancePaise
}

func TestBalanceFollowsIncomeExpenseAndVoid(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if got := f.balance(t); got != 100_000 {
		t.Fatalf("opening balance = %d", got)
	}
	salary, _, err := f.svc.CreateTransaction(ctx, f.tx(Income, 50_050, "k1"))
	if err != nil {
		t.Fatal(err)
	}
	rent, _, err := f.svc.CreateTransaction(ctx, f.tx(Expense, 30_025, "k2"))
	if err != nil {
		t.Fatal(err)
	}
	if got := f.balance(t); got != 100_000+50_050-30_025 {
		t.Fatalf("balance = %d", got)
	}
	if _, err := f.svc.VoidTransaction(ctx, rent.ID, "  entered twice "); err != nil {
		t.Fatal(err)
	}
	if got := f.balance(t); got != 150_050 {
		t.Fatalf("balance after void = %d", got)
	}
	if _, err := f.svc.VoidTransaction(ctx, salary.ID, ""); err != nil {
		t.Fatal(err)
	}
	if got := f.balance(t); got != 100_000 {
		t.Fatalf("balance with everything void = %d", got)
	}
}

func TestVoidKeepsTheRecordAndTimestamps(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	created, _, _ := f.svc.CreateTransaction(ctx, f.tx(Expense, 100, "k1"))
	voided, err := f.svc.VoidTransaction(ctx, created.ID, " wrong account ")
	if err != nil {
		t.Fatal(err)
	}
	if !voided.Voided() || voided.VoidReason != "wrong account" || !voided.VoidedAt.Equal(clockUTC) {
		t.Fatalf("voided = %+v", voided)
	}
	list, _ := f.svc.ListTransactions(ctx, f.account.ID)
	if len(list) != 1 || !list[0].Voided() {
		t.Fatalf("list = %+v", list)
	}
}

func TestVoidErrors(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	created, _, _ := f.svc.CreateTransaction(ctx, f.tx(Expense, 100, "k1"))
	if _, err := f.svc.VoidTransaction(ctx, created.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.VoidTransaction(ctx, created.ID, ""); !errors.Is(err, ErrAlreadyVoided) {
		t.Errorf("second void err = %v", err)
	}
	if _, err := f.svc.VoidTransaction(ctx, uuid.NewV7(), ""); !errors.Is(err, ErrTransactionNotFound) {
		t.Errorf("unknown err = %v", err)
	}
	if _, err := f.svc.VoidTransaction(ctx, created.ID, strings.Repeat("x", MaxReasonLength+1)); !errors.Is(err, ErrInvalidReason) {
		t.Errorf("long reason err = %v", err)
	}
}

func TestAmountValidation(t *testing.T) {
	f := newFixture(t)
	cases := map[string]int64{
		"zero": 0, "negative": -5, "over cap": MaxAmountPaise + 1,
	}
	for name, amount := range cases {
		_, _, err := f.svc.CreateTransaction(context.Background(), f.tx(Expense, amount, "k-"+strings.ReplaceAll(name, " ", "-")))
		if !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	for name, amount := range map[string]int64{"one paisa": 1, "cap": MaxAmountPaise} {
		if _, _, err := f.svc.CreateTransaction(context.Background(), f.tx(Expense, amount, "ok-"+strings.ReplaceAll(name, " ", "-"))); err != nil {
			t.Errorf("%s should be accepted: %v", name, err)
		}
	}
}

func TestOpeningBalanceLimits(t *testing.T) {
	svc := NewService(newMemoryRepo(), func() time.Time { return clockUTC })
	ctx := context.Background()
	if _, err := svc.CreateAccount(ctx, "Overdraft", -5_000); err != nil {
		t.Errorf("negative opening balance should be allowed: %v", err)
	}
	if _, err := svc.CreateAccount(ctx, "Zero", 0); err != nil {
		t.Errorf("zero opening balance should be allowed: %v", err)
	}
	for _, v := range []int64{MaxAmountPaise + 1, -MaxAmountPaise - 1} {
		if _, err := svc.CreateAccount(ctx, "Huge", v); !errors.Is(err, ErrInvalidOpening) {
			t.Errorf("opening %d err = %v", v, err)
		}
	}
}

func TestDateHandling(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	for _, bad := range []string{"2026-02-30", "2026-13-01", "01-10-2026", "2026-1-5", "yesterday", "2026-10-01T00:00:00Z"} {
		in := f.tx(Expense, 100, "bad-date")
		in.OccurredOn = bad
		if _, _, err := f.svc.CreateTransaction(ctx, in); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("date %q err = %v", bad, err)
		}
	}
	leap := f.tx(Expense, 100, "leap")
	leap.OccurredOn = "2028-02-29"
	if tx, _, err := f.svc.CreateTransaction(ctx, leap); err != nil || tx.OccurredOn.Format(DateLayout) != "2028-02-29" {
		t.Errorf("leap day: %+v %v", tx, err)
	}
	notLeap := f.tx(Expense, 100, "notleap")
	notLeap.OccurredOn = "2027-02-29"
	if _, _, err := f.svc.CreateTransaction(ctx, notLeap); !errors.Is(err, ErrInvalidDate) {
		t.Errorf("2027-02-29 err = %v", err)
	}
}

func TestDefaultDateIsTodayInIndia(t *testing.T) {
	f := newFixture(t)
	in := f.tx(Expense, 100, "today")
	in.OccurredOn = ""
	tx, _, err := f.svc.CreateTransaction(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if got := tx.OccurredOn.Format(DateLayout); got != "2026-10-10" {
		t.Fatalf("20:00 UTC on Oct 9 is Oct 10 in India; got %s", got)
	}
	if !tx.CreatedAt.Equal(clockUTC) || tx.CreatedAt.Location() != time.UTC {
		t.Fatalf("createdAt = %v", tx.CreatedAt)
	}
}

func TestMonthEndBoundaryInIndia(t *testing.T) {
	svc := NewService(newMemoryRepo(), func() time.Time { return time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC) })
	if got := svc.Today().Format(DateLayout); got != "2026-11-01" {
		t.Fatalf("19:00 UTC on Oct 31 is Nov 1 in India; got %s", got)
	}
	earlier := NewService(newMemoryRepo(), func() time.Time { return time.Date(2026, 10, 31, 18, 29, 59, 0, time.UTC) })
	if got := earlier.Today().Format(DateLayout); got != "2026-10-31" {
		t.Fatalf("18:29:59 UTC on Oct 31 is still Oct 31 in India; got %s", got)
	}
}

func TestIdempotency(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first, created, err := f.svc.CreateTransaction(ctx, f.tx(Expense, 500, "same"))
	if err != nil || !created {
		t.Fatalf("first: %v created=%v", err, created)
	}
	again, created, err := f.svc.CreateTransaction(ctx, f.tx(Expense, 500, "same"))
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("replay: %+v created=%v err=%v", again, created, err)
	}
	if got := f.balance(t); got != 100_000-500 {
		t.Fatalf("balance after replay = %d", got)
	}
	if _, _, err := f.svc.CreateTransaction(ctx, f.tx(Expense, 501, "same")); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("different payload err = %v", err)
	}
	if _, _, err := f.svc.CreateTransaction(ctx, f.tx(Expense, 500, "other")); err != nil {
		t.Fatalf("a new key is a new transaction: %v", err)
	}
	if got := f.balance(t); got != 100_000-1_000 {
		t.Fatalf("balance = %d", got)
	}
}

func TestIdempotencyKeyRules(t *testing.T) {
	f := newFixture(t)
	for _, key := range []string{"", "has space", "semi;colon", strings.Repeat("a", MaxKeyLength+1), "é"} {
		if _, _, err := f.svc.CreateTransaction(context.Background(), f.tx(Expense, 100, key)); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("key %q err = %v", key, err)
		}
	}
	if _, _, err := f.svc.CreateTransaction(context.Background(), f.tx(Expense, 100, strings.Repeat("a", MaxKeyLength))); err != nil {
		t.Errorf("max length key: %v", err)
	}
}

func TestConcurrentSameKeyCreatesOneTransaction(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = f.svc.CreateTransaction(context.Background(), f.tx(Expense, 700, "race"))
		}()
	}
	wg.Wait()
	if got := f.balance(t); got != 100_000-700 {
		t.Fatalf("balance = %d, want one expense of 700", got)
	}
}

func TestKindAndReferenceRules(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	mismatch := f.tx(Expense, 100, "m")
	mismatch.CategoryID = f.income.ID
	if _, _, err := f.svc.CreateTransaction(ctx, mismatch); !errors.Is(err, ErrKindMismatch) {
		t.Errorf("mismatch err = %v", err)
	}
	badKind := f.tx(Expense, 100, "b")
	badKind.Kind = "transfer"
	if _, _, err := f.svc.CreateTransaction(ctx, badKind); !errors.Is(err, ErrInvalidKind) {
		t.Errorf("kind err = %v", err)
	}
	noAccount := f.tx(Expense, 100, "a")
	noAccount.AccountID = uuid.NewV7()
	if _, _, err := f.svc.CreateTransaction(ctx, noAccount); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("account err = %v", err)
	}
	noCategory := f.tx(Expense, 100, "c")
	noCategory.CategoryID = uuid.NewV7()
	if _, _, err := f.svc.CreateTransaction(ctx, noCategory); !errors.Is(err, ErrCategoryNotFound) {
		t.Errorf("category err = %v", err)
	}
}

func TestNamesAndCategories(t *testing.T) {
	svc := NewService(newMemoryRepo(), func() time.Time { return clockUTC })
	ctx := context.Background()
	if _, err := svc.CreateAccount(ctx, "   ", 0); !errors.Is(err, ErrInvalidName) {
		t.Errorf("blank err = %v", err)
	}
	if _, err := svc.CreateAccount(ctx, strings.Repeat("₹", MaxNameLength), 0); err != nil {
		t.Errorf("200 multi-byte characters: %v", err)
	}
	if _, err := svc.CreateAccount(ctx, "Main", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateAccount(ctx, "  main ", 0); !errors.Is(err, ErrDuplicateName) {
		t.Errorf("duplicate account err = %v", err)
	}
	if _, err := svc.CreateCategory(ctx, "Salary", Income, true); !errors.Is(err, ErrCommittedIncome) {
		t.Errorf("committed income err = %v", err)
	}
	if _, err := svc.CreateCategory(ctx, "Food", "other", false); !errors.Is(err, ErrInvalidKind) {
		t.Errorf("kind err = %v", err)
	}
	if _, err := svc.CreateCategory(ctx, "Food", Expense, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategory(ctx, "food", Expense, false); !errors.Is(err, ErrDuplicateName) {
		t.Errorf("duplicate category err = %v", err)
	}
	if _, err := svc.CreateCategory(ctx, "Food", Income, false); err != nil {
		t.Errorf("same name under a different kind is allowed: %v", err)
	}
}

func TestSignedPaise(t *testing.T) {
	if SignedPaise(Income, 5) != 5 || SignedPaise(Expense, 5) != -5 {
		t.Fatal("sign convention broken")
	}
}

func TestCategoryUpdateAndArchive(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	cat, err := f.svc.CreateCategory(ctx, "Gaming", Expense, false)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := f.svc.UpdateCategory(ctx, cat.ID, "Video Games", Expense, false, true)
	if err != nil {
		t.Fatalf("update category: %v", err)
	}
	if updated.Name != "Video Games" || !updated.Archived {
		t.Fatalf("expected updated fields, got %+v", updated)
	}

	// Delete category without transactions should succeed
	if err := f.svc.DeleteCategory(ctx, cat.ID); err != nil {
		t.Fatalf("delete unused category: %v", err)
	}

	// Delete category with transactions should fail with ErrCategoryInUse
	f.spend(t, f.expense, 500, "2026-10-02", "spend-rent")
	err = f.svc.DeleteCategory(ctx, f.expense.ID)
	if !errors.Is(err, ErrCategoryInUse) {
		t.Fatalf("expected ErrCategoryInUse, got: %v", err)
	}

	// But archiving it should succeed
	archived, err := f.svc.UpdateCategory(ctx, f.expense.ID, f.expense.Name, f.expense.Kind, f.expense.Committed, true)
	if err != nil || !archived.Archived {
		t.Fatalf("expected archived category, got %+v, err: %v", archived, err)
	}
}

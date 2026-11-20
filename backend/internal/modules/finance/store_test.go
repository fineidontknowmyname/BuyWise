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

func storeFixture(t *testing.T) (*Service, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST is not set; skipping database test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := migrate.Apply(ctx, pool, os.DirFS("../../../migrations")); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE finance.transactions, finance.accounts CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return NewService(NewStore(pool), func() time.Time { return clockUTC }), pool
}

func categoryNamed(t *testing.T, svc *Service, name string) Category {
	t.Helper()
	list, err := svc.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("category %q not found", name)
	return Category{}
}

func TestStoreSeedsStarterCategories(t *testing.T) {
	svc, _ := storeFixture(t)
	list, err := svc.ListCategories(context.Background())
	if err != nil || len(list) < 12 {
		t.Fatalf("categories = %d, err %v", len(list), err)
	}
	if rent := categoryNamed(t, svc, "Rent"); rent.Kind != Expense || !rent.Committed {
		t.Errorf("rent = %+v", rent)
	}
	if salary := categoryNamed(t, svc, "Salary"); salary.Kind != Income || salary.Committed {
		t.Errorf("salary = %+v", salary)
	}
}

func TestStoreBalanceIdempotencyAndVoid(t *testing.T) {
	svc, _ := storeFixture(t)
	ctx := context.Background()
	account, err := svc.CreateAccount(ctx, "Savings", 100_000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateAccount(ctx, "SAVINGS", 0); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("duplicate err = %v", err)
	}
	salary, rent := categoryNamed(t, svc, "Salary"), categoryNamed(t, svc, "Rent")

	in := NewTransaction{AccountID: account.ID, CategoryID: salary.ID, Kind: Income, AmountPaise: 50_050, OccurredOn: "2026-10-01", IdempotencyKey: "a"}
	first, created, err := svc.CreateTransaction(ctx, in)
	if err != nil || !created {
		t.Fatalf("create: %v %v", err, created)
	}
	replay, created, err := svc.CreateTransaction(ctx, in)
	if err != nil || created || replay.ID != first.ID {
		t.Fatalf("replay: %+v created=%v err=%v", replay, created, err)
	}
	changed := in
	changed.AmountPaise = 1
	if _, _, err := svc.CreateTransaction(ctx, changed); !errors.Is(err, ErrKeyConflict) {
		t.Fatalf("conflict err = %v", err)
	}
	expense, _, err := svc.CreateTransaction(ctx, NewTransaction{AccountID: account.ID, CategoryID: rent.ID, Kind: Expense, AmountPaise: 30_025, OccurredOn: "2026-10-02", IdempotencyKey: "b"})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := svc.GetAccount(ctx, account.ID)
	if got.BalancePaise != 100_000+50_050-30_025 {
		t.Fatalf("balance = %d", got.BalancePaise)
	}
	if first.OccurredOn.Format(DateLayout) != "2026-10-01" {
		t.Fatalf("stored date = %v", first.OccurredOn)
	}

	voided, err := svc.VoidTransaction(ctx, expense.ID, "wrong")
	if err != nil || !voided.Voided() || voided.VoidReason != "wrong" {
		t.Fatalf("void: %+v %v", voided, err)
	}
	if _, err := svc.VoidTransaction(ctx, expense.ID, ""); !errors.Is(err, ErrAlreadyVoided) {
		t.Fatalf("second void err = %v", err)
	}
	if _, err := svc.VoidTransaction(ctx, uuid.NewV7(), ""); !errors.Is(err, ErrTransactionNotFound) {
		t.Fatalf("unknown void err = %v", err)
	}
	got, _ = svc.GetAccount(ctx, account.ID)
	if got.BalancePaise != 150_050 {
		t.Fatalf("balance after void = %d", got.BalancePaise)
	}

	list, err := svc.ListTransactions(ctx, account.ID)
	if err != nil || len(list) != 2 || list[0].ID != expense.ID || !list[0].Voided() {
		t.Fatalf("list = %+v err %v (newest date first, voided kept)", list, err)
	}
}

func TestStoreAccountWithNoTransactions(t *testing.T) {
	svc, _ := storeFixture(t)
	ctx := context.Background()
	account, _ := svc.CreateAccount(ctx, "Empty", -2_500)
	got, err := svc.GetAccount(ctx, account.ID)
	if err != nil || got.BalancePaise != -2_500 {
		t.Fatalf("got %+v err %v", got, err)
	}
	if _, err := svc.GetAccount(ctx, uuid.NewV7()); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("unknown err = %v", err)
	}
	list, err := svc.ListTransactions(ctx, account.ID)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("list = %v err %v", list, err)
	}
}

func TestStoreRejectsOutOfRangeValues(t *testing.T) {
	_, pool := storeFixture(t)
	ctx := context.Background()
	id := uuid.NewV7().String()
	if _, err := pool.Exec(ctx, `INSERT INTO finance.accounts (id, name, opening_balance_paise, created_at) VALUES ($1::uuid, 'X', 1000000000001, now())`, id); err == nil {
		t.Fatal("database should reject an opening balance over the cap")
	}
}

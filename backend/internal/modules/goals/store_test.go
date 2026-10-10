package goals

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

func testPool(t *testing.T) *pgxpool.Pool {
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
	if _, err := pool.Exec(ctx, "TRUNCATE goals.goals"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

func TestStoreRoundTrip(t *testing.T) {
	store := NewStore(testPool(t))
	ctx := context.Background()
	created := time.Date(2026, 10, 9, 10, 0, 0, 123000, time.UTC)

	first := Goal{ID: uuid.NewV7(), Name: "Laptop", CreatedAt: created}
	second := Goal{ID: uuid.NewV7(), Name: "Fridge", CreatedAt: created}
	for _, g := range []Goal{first, second} {
		if err := store.Create(ctx, g); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	got, err := store.Get(ctx, first.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != first {
		t.Fatalf("got %+v, want %+v", got, first)
	}

	list, err := store.List(ctx)
	if err != nil || len(list) != 2 || list[0].ID != first.ID || list[1].ID != second.ID {
		t.Fatalf("list = %+v, err %v", list, err)
	}
}

func TestStoreGetMissing(t *testing.T) {
	store := NewStore(testPool(t))
	if _, err := store.Get(context.Background(), uuid.NewV7()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestStoreRejectsDuplicateID(t *testing.T) {
	store := NewStore(testPool(t))
	ctx := context.Background()
	g := Goal{ID: uuid.NewV7(), Name: "Laptop", CreatedAt: time.Now().UTC()}
	if err := store.Create(ctx, g); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, g); err == nil {
		t.Fatal("expected duplicate ID to fail")
	}
}

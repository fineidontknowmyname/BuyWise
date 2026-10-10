package migrate

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func isolatedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		t.Skip("DATABASE_URL_TEST is not set; skipping database test")
	}
	ctx := context.Background()

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	schema := fmt.Sprintf("test_migrate_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect isolated: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestApplyRunsInOrderAndIsRepeatable(t *testing.T) {
	pool := isolatedPool(t)
	ctx := context.Background()
	files := fstest.MapFS{
		"0002_second.sql": {Data: []byte("ALTER TABLE probe ADD COLUMN note text;")},
		"0001_first.sql":  {Data: []byte("CREATE TABLE probe (id int PRIMARY KEY);")},
		"readme.txt":      {Data: []byte("ignored")},
	}

	applied, err := Apply(ctx, pool, files)
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if !slices.Equal(applied, []string{"0001_first.sql", "0002_second.sql"}) {
		t.Fatalf("applied = %v", applied)
	}

	again, err := Apply(ctx, pool, files)
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second apply re-ran %v", again)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count history: %v", err)
	}
	if count != 2 {
		t.Fatalf("history rows = %d, want 2", count)
	}
}

func TestApplyStopsAndRollsBackOnFailure(t *testing.T) {
	pool := isolatedPool(t)
	ctx := context.Background()
	files := fstest.MapFS{
		"0001_good.sql": {Data: []byte("CREATE TABLE good (id int);")},
		"0002_bad.sql":  {Data: []byte("CREATE TABLE half (id int); THIS IS NOT SQL;")},
		"0003_next.sql": {Data: []byte("CREATE TABLE never (id int);")},
	}

	applied, err := Apply(ctx, pool, files)
	if err == nil {
		t.Fatal("expected an error from the bad migration")
	}
	if !slices.Equal(applied, []string{"0001_good.sql"}) {
		t.Fatalf("applied = %v", applied)
	}

	var exists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('half') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatalf("check table: %v", err)
	}
	if exists {
		t.Fatal("partial migration was not rolled back")
	}
}

func TestApplyWithNoMigrations(t *testing.T) {
	pool := isolatedPool(t)
	applied, err := Apply(context.Background(), pool, fstest.MapFS{})
	if err != nil || len(applied) != 0 {
		t.Fatalf("applied = %v, err = %v", applied, err)
	}
}

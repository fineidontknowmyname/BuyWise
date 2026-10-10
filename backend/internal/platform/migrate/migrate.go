package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const createHistoryTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version text PRIMARY KEY,
	applied_at timestamptz NOT NULL DEFAULT now()
)`

func Apply(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) ([]string, error) {
	if _, err := pool.Exec(ctx, createHistoryTable); err != nil {
		return nil, fmt.Errorf("create migration history: %w", err)
	}
	files, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	var applied []string
	for _, file := range files {
		version := path.Base(file)
		done, err := applyOne(ctx, pool, migrations, file, version)
		if err != nil {
			return applied, err
		}
		if done {
			applied = append(applied, version)
		}
	}
	return applied, nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS, file, version string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existing string
	err = tx.QueryRow(ctx, `SELECT version FROM schema_migrations WHERE version = $1`, version).Scan(&existing)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("check %s: %w", version, err)
	}

	body, err := fs.ReadFile(migrations, file)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return false, fmt.Errorf("apply %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return false, fmt.Errorf("record %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit %s: %w", version, err)
	}
	return true, nil
}

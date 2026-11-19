package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type tenantKey struct{}

type Queries interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

func WithTenantTransaction(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, tenantKey{}, tx)
}

func Scoped(ctx context.Context, fallback Queries) Queries {
	if tx, ok := ctx.Value(tenantKey{}).(pgx.Tx); ok {
		return tx
	}
	return fallback
}

func LockPlanning(ctx context.Context, fallback Queries) error {
	_, err := Scoped(ctx, fallback).Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('buywise.planning:' || identity.current_user_id()::text, 0))`)
	return err
}

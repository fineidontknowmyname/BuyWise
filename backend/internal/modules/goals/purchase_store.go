package goals

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"selfemi/internal/platform/database"
)

type PurchaseEvent struct {
	Key       string
	Details   []byte
	CreatedAt time.Time
}

func (s *Store) Purchase(ctx context.Context, id uuid.UUID) (*PurchaseEvent, error) {
	var event PurchaseEvent
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT request_key::text, details, created_at
		FROM goals.purchases WHERE goal_id = $1::uuid`, id.String()).Scan(&event.Key, &event.Details, &event.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &event, err
}

func (s *Store) CompletePurchase(ctx context.Context, id uuid.UUID, key, expense string, details []byte, surplus map[string]int64) error {
	db := database.Scoped(ctx, s.pool)
	_, err := db.Exec(ctx, `INSERT INTO goals.purchases (goal_id, request_key, transaction_id, details)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::jsonb)`, id.String(), key, expense, details)
	if err != nil {
		return err
	}
	for account, amount := range surplus {
		if _, err := db.Exec(ctx, `INSERT INTO goals.purchase_surplus (goal_id, account_id, amount_paise)
			VALUES ($1::uuid, $2::uuid, $3)`, id.String(), account, amount); err != nil {
			return err
		}
	}
	return s.setLifecycle(ctx, id, "completed")
}

func (s *Store) RemainingSurplus(ctx context.Context, id uuid.UUID) (map[string]int64, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT account_id::text, amount_paise
		FROM goals.remaining_surplus WHERE goal_id = $1::uuid`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]int64{}
	for rows.Next() {
		var account string
		var amount int64
		if err := rows.Scan(&account, &amount); err != nil {
			return nil, err
		}
		result[account] = amount
	}
	return result, rows.Err()
}

func (s *Store) SurplusAllocations(ctx context.Context, id uuid.UUID, key string) ([]PurchaseEvent, error) {
	query := `SELECT request_key::text, details, created_at FROM goals.surplus_allocations WHERE goal_id = $1::uuid`
	args := []any{id.String()}
	if key != "" {
		query += ` AND request_key = $2::uuid`
		args = append(args, key)
	}
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, query+` ORDER BY created_at DESC, request_key DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []PurchaseEvent{}
	for rows.Next() {
		var event PurchaseEvent
		if err := rows.Scan(&event.Key, &event.Details, &event.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, rows.Err()
}

func (s *Store) RecordSurplusAllocation(ctx context.Context, id uuid.UUID, key string, details []byte, debits map[string]int64) error {
	db := database.Scoped(ctx, s.pool)
	if _, err := db.Exec(ctx, `INSERT INTO goals.surplus_allocations (goal_id, request_key, details)
		VALUES ($1::uuid, $2::uuid, $3::jsonb)`, id.String(), key, details); err != nil {
		return err
	}
	for account, amount := range debits {
		if _, err := db.Exec(ctx, `INSERT INTO goals.surplus_debits (goal_id, request_key, account_id, amount_paise)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4)`, id.String(), key, account, amount); err != nil {
			return err
		}
	}
	return nil
}

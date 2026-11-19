package goals

import (
	"context"
	"errors"
	"fmt"
	"selfemi/internal/platform/database"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, goal Goal) error {
	var prodID *string
	if goal.ProductID != nil {
		str := goal.ProductID.String()
		prodID = &str
	}
	priority := goal.Priority
	if priority == "" {
		priority = "medium"
	}
	status := goal.Status
	if status == "" {
		status = "active"
	}

	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO goals.goals (
			id, name, target_price_paise, saved_amount_paise, target_date, priority, product_id, status, created_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5::date, $6, $7::uuid, $8, $9
		)`,
		goal.ID.String(), goal.Name, goal.TargetPricePaise, goal.SavedAmountPaise,
		goal.TargetDate, priority, prodID, status, goal.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create goal: %w", err)
	}
	return nil
}

func (s *Store) Update(ctx context.Context, goal Goal) error {
	var prodID *string
	if goal.ProductID != nil {
		str := goal.ProductID.String()
		prodID = &str
	}
	priority := goal.Priority
	if priority == "" {
		priority = "medium"
	}
	status := goal.Status
	if status == "" {
		status = "active"
	}

	cmd, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`UPDATE goals.goals SET
			name = $2, target_price_paise = $3, saved_amount_paise = $4,
			target_date = $5::date, priority = $6, product_id = $7::uuid, status = $8
		WHERE id = $1::uuid`,
		goal.ID.String(), goal.Name, goal.TargetPricePaise, goal.SavedAmountPaise,
		goal.TargetDate, priority, prodID, status,
	)
	if err != nil {
		return fmt.Errorf("update goal: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (Goal, error) {
	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT
			id::text, name, target_price_paise, saved_amount_paise,
			target_date::text, priority, product_id::text, status, created_at
		FROM goals.goals WHERE id = $1::uuid`,
		id.String(),
	)
	goal, err := scanGoal(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Goal{}, ErrNotFound
	}
	if err != nil {
		return Goal{}, fmt.Errorf("get goal: %w", err)
	}
	return goal, nil
}

func (s *Store) List(ctx context.Context) ([]Goal, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT
			id::text, name, target_price_paise, saved_amount_paise,
			target_date::text, priority, product_id::text, status, created_at
		FROM goals.goals ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()

	var list []Goal
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan goal: %w", err)
		}
		list = append(list, goal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list goals rows: %w", err)
	}
	return list, nil
}

func scanGoal(row pgx.Row) (Goal, error) {
	var goal Goal
	var idStr string
	var prodStr *string
	var targetDate *string
	err := row.Scan(
		&idStr, &goal.Name, &goal.TargetPricePaise, &goal.SavedAmountPaise,
		&targetDate, &goal.Priority, &prodStr, &goal.Status, &goal.CreatedAt,
	)
	if err != nil {
		return Goal{}, err
	}
	parsed, err := uuid.Parse(idStr)
	if err != nil {
		return Goal{}, fmt.Errorf("parse goal id: %w", err)
	}
	goal.ID = parsed
	goal.TargetDate = targetDate
	if prodStr != nil {
		pID, _ := uuid.Parse(*prodStr)
		goal.ProductID = &pID
	}
	goal.CreatedAt = goal.CreatedAt.UTC()
	return goal, nil
}

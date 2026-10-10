package goals

import (
	"context"
	"errors"
	"fmt"
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
	_, err := s.pool.Exec(ctx,
		`INSERT INTO goals.goals (id, name, created_at) VALUES ($1::uuid, $2, $3)`,
		goal.ID.String(), goal.Name, goal.CreatedAt)
	if err != nil {
		return fmt.Errorf("create goal: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id uuid.UUID) (Goal, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id::text, name, created_at FROM goals.goals WHERE id = $1::uuid`, id.String())
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
	rows, err := s.pool.Query(ctx, `SELECT id::text, name, created_at FROM goals.goals ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	defer rows.Close()

	list := []Goal{}
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("list goals: %w", err)
		}
		list = append(list, goal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}
	return list, nil
}

func scanGoal(row pgx.Row) (Goal, error) {
	var goal Goal
	var id string
	if err := row.Scan(&id, &goal.Name, &goal.CreatedAt); err != nil {
		return Goal{}, err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return Goal{}, fmt.Errorf("parse goal id: %w", err)
	}
	goal.ID = parsed
	goal.CreatedAt = goal.CreatedAt.UTC()
	return goal, nil
}

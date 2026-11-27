package decision

import (
	"context"
	"encoding/json"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/platform/database"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) lock(ctx context.Context) error {
	return database.LockPlanning(ctx, s.pool)
}

func (s *Store) save(ctx context.Context, value Decision) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO decision.purchase_decisions
        (id, goal_id, decided_at, rules_version, state, payload) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::jsonb)`,
		value.ID, value.GoalID, value.DecidedAt, value.RulesVersion, value.State, payload)
	return err
}

func (s *Store) history(ctx context.Context, goalID uuid.UUID) ([]Decision, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT payload FROM decision.purchase_decisions
        WHERE goal_id = $1::uuid ORDER BY decided_at DESC, id DESC LIMIT 20`, goalID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []Decision{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var record Decision
		if err := json.Unmarshal(raw, &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

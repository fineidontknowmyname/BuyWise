package goals

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"selfemi/internal/platform/database"
)

func (s *Store) lockPlanning(ctx context.Context) error {
	return database.LockPlanning(ctx, s.pool)
}

func (s *Store) planCandidates(ctx context.Context) ([]planCandidate, error) {
	goals, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]planCandidate, len(goals))
	positions := make(map[string]int, len(goals))
	for i, goal := range goals {
		result[i] = planCandidate{Goal: goal, Terms: defaultTerms(), FundedPaise: goal.AllocatedPaise}
		positions[goal.ID.String()] = i
	}
	db := database.Scoped(ctx, s.pool)
	rows, err := db.Query(ctx, `SELECT DISTINCT ON (goal_id) goal_id::text, revision, mode, frequency, approved_paise
        FROM goals.plan_revisions ORDER BY goal_id, revision DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var terms planTerms
		if err := rows.Scan(&id, &terms.Revision, &terms.Mode, &terms.Frequency, &terms.ApprovedPaise); err != nil {
			return nil, err
		}
		if i, exists := positions[id]; exists {
			result[i].Terms = terms
		}
	}
	return result, rows.Err()
}

func (s *Store) planHistory(ctx context.Context, id uuid.UUID) ([]planTerms, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT revision, mode, frequency, approved_paise
        FROM goals.plan_revisions WHERE goal_id = $1::uuid ORDER BY revision DESC LIMIT 100`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []planTerms{}
	for rows.Next() {
		var record planTerms
		if err := rows.Scan(&record.Revision, &record.Mode, &record.Frequency, &record.ApprovedPaise); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *Store) saveTerms(ctx context.Context, id uuid.UUID, terms planTerms) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO goals.plan_revisions
        (goal_id, revision, mode, frequency, approved_paise) VALUES ($1::uuid, $2, $3, $4, $5)`,
		id.String(), terms.Revision+1, terms.Mode, terms.Frequency, terms.ApprovedPaise)
	return err
}

func (s *Store) fundingHistory(ctx context.Context, id uuid.UUID) ([]fundingEntry, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT id::text, account_id::text, amount_paise, created_at
        FROM goals.funding WHERE goal_id = $1::uuid ORDER BY created_at DESC, id DESC LIMIT 100`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []fundingEntry{}
	for rows.Next() {
		var entry fundingEntry
		if err := rows.Scan(&entry.ID, &entry.AccountID, &entry.AmountPaise, &entry.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *Store) reservations(ctx context.Context) (map[string]int64, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT f.account_id::text, SUM(f.amount_paise)::bigint
        FROM goals.reserved_funds f GROUP BY f.account_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]int64{}
	for rows.Next() {
		var id string
		var amount int64
		if err := rows.Scan(&id, &amount); err != nil {
			return nil, err
		}
		result[id] = amount
	}
	return result, rows.Err()
}

type AccountReservations struct {
	ForGoal    int64
	OtherGoals int64
}

func (s *Store) PurchaseReservations(ctx context.Context, id uuid.UUID) (map[string]AccountReservations, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT f.account_id::text,
        COALESCE(SUM(f.amount_paise) FILTER (WHERE f.goal_id = $1::uuid AND NOT f.is_surplus), 0)::bigint,
        COALESCE(SUM(f.amount_paise) FILTER (WHERE f.goal_id <> $1::uuid OR f.is_surplus), 0)::bigint
        FROM goals.reserved_funds f GROUP BY f.account_id`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]AccountReservations{}
	for rows.Next() {
		var account string
		var item AccountReservations
		if err := rows.Scan(&account, &item.ForGoal, &item.OtherGoals); err != nil {
			return nil, err
		}
		result[account] = item
	}
	return result, rows.Err()
}

func (s *Store) priorFunding(ctx context.Context, id uuid.UUID, input fundingInput) (bool, error) {
	var goalID, accountID string
	var amount int64
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT goal_id::text, account_id::text, amount_paise
        FROM goals.funding WHERE idempotency_key = $1`, input.IdempotencyKey).Scan(&goalID, &accountID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if goalID != id.String() || accountID != input.AccountID || amount != input.AmountPaise {
		return false, errFundingConflict
	}
	return true, nil
}

func (s *Store) addFunding(ctx context.Context, id uuid.UUID, input fundingInput) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO goals.funding
        (goal_id, account_id, amount_paise, idempotency_key) VALUES ($1::uuid, $2::uuid, $3, $4)`,
		id.String(), input.AccountID, input.AmountPaise, input.IdempotencyKey)
	return err
}

func (s *Store) AddSurplusFunding(ctx context.Context, id uuid.UUID, accountID string, amount int64) error {
	return s.addFunding(ctx, id, fundingInput{AccountID: accountID, AmountPaise: amount, IdempotencyKey: uuid.NewV7().String()})
}

func (s *Store) setLifecycle(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx, `UPDATE goals.goals SET status = $2
        WHERE id = $1::uuid AND (status IN ('active', 'paused') OR status = $2)`, id.String(), status)
	if err == nil && tag.RowsAffected() == 0 {
		return errPlanUnavailable
	}
	return err
}

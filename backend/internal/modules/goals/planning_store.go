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
	rows, err := db.Query(ctx, `SELECT DISTINCT ON (p.goal_id) p.goal_id::text, p.revision, p.mode, p.frequency, p.approved_paise,
        COALESCE(p.first_saving_on::text, ''), p.created_at,
        COALESCE((SELECT SUM(f.amount_paise)::bigint FROM goals.funding f WHERE f.goal_id=p.goal_id AND f.funding_kind='contribution' AND f.created_at>=p.created_at),0)
        FROM goals.plan_revisions p ORDER BY p.goal_id, p.revision DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var terms planTerms
		var paid int64
		if err := rows.Scan(&id, &terms.Revision, &terms.Mode, &terms.Frequency, &terms.ApprovedPaise, &terms.FirstSavingOn, &terms.SavedAt, &paid); err != nil {
			return nil, err
		}
		if i, exists := positions[id]; exists {
			result[i].Terms = terms
			result[i].RegularPaidPaise = paid
		}
	}
	return result, rows.Err()
}

func (s *Store) planHistory(ctx context.Context, id uuid.UUID) ([]planTerms, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT revision, mode, frequency, approved_paise, COALESCE(first_saving_on::text,''), created_at
        FROM goals.plan_revisions WHERE goal_id = $1::uuid ORDER BY revision DESC LIMIT 100`, id.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []planTerms{}
	for rows.Next() {
		var record planTerms
		if err := rows.Scan(&record.Revision, &record.Mode, &record.Frequency, &record.ApprovedPaise, &record.FirstSavingOn, &record.SavedAt); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *Store) saveTerms(ctx context.Context, id uuid.UUID, terms planTerms) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO goals.plan_revisions
        (goal_id, revision, mode, frequency, approved_paise, first_saving_on) VALUES ($1::uuid, $2, $3, $4, $5, NULLIF($6,'')::date)`,
		id.String(), terms.Revision+1, terms.Mode, terms.Frequency, terms.ApprovedPaise, terms.FirstSavingOn)
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

func (s *Store) AccountReservations(ctx context.Context) (map[string]int64, error) {
	return s.reservations(ctx)
}

func (s *Store) MonthlyCommitments(ctx context.Context) (int64, error) {
	candidates, err := s.planCandidates(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, candidate := range candidates {
		if candidate.Goal.Status == "active" && candidate.Goal.TargetPricePaise > candidate.FundedPaise {
			total += (candidate.Terms.ApprovedPaise*periodsPerYear(candidate.Terms.Frequency) + 11) / 12
		}
	}
	return total, nil
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
	var goalID, accountID, kind string
	var amount int64
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT goal_id::text, account_id::text, amount_paise, funding_kind
        FROM goals.funding WHERE idempotency_key = $1`, input.IdempotencyKey).Scan(&goalID, &accountID, &amount, &kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if goalID != id.String() || accountID != input.AccountID || amount != input.AmountPaise || kind != input.Kind {
		return false, errFundingConflict
	}
	return true, nil
}

func (s *Store) addFunding(ctx context.Context, id uuid.UUID, input fundingInput) error {
	if input.Kind == "" {
		input.Kind = "topup"
	}
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO goals.funding
        (goal_id, account_id, amount_paise, idempotency_key, funding_kind) VALUES ($1::uuid, $2::uuid, $3, $4, $5)`,
		id.String(), input.AccountID, input.AmountPaise, input.IdempotencyKey, input.Kind)
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

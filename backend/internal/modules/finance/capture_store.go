package finance

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"selfemi/internal/platform/database"
)

func (s *Store) lockCapture(ctx context.Context) error {
	return database.LockPlanning(ctx, s.pool)
}

func (s *Store) transactionBy(ctx context.Context, column, value string) (*Transaction, error) {
	condition := "id = $1::uuid"
	if column == "idempotency_key" {
		condition = "idempotency_key = $1"
	}
	row := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT `+transactionColumns+` FROM finance.transactions WHERE `+condition, value)
	item, err := scanTransaction(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &item, err
}

const candidateColumns = `id::text, source_app, merchant, amount_paise, occurred_on::text, status, transaction_id::text`

func scanCandidate(row pgx.Row, includeDuplicate bool) (captureCandidate, error) {
	var item captureCandidate
	fields := []any{&item.ID, &item.SourceApp, &item.Merchant, &item.AmountPaise, &item.OccurredOn, &item.Status, &item.TransactionID}
	if includeDuplicate {
		fields = append(fields, &item.PossibleDuplicate)
	}
	err := row.Scan(fields...)
	return item, err
}

func (s *Store) ingestCandidate(ctx context.Context, input captureCandidate, fingerprint string, reference *string) (captureCandidate, bool, error) {
	db := database.Scoped(ctx, s.pool)
	row := db.QueryRow(ctx, `SELECT `+candidateColumns+` FROM finance.expense_candidates
		WHERE (source_app = $1 AND reference_hash = $2) OR (fingerprint = $3 AND (reference_hash IS NULL OR $2::text IS NULL))
		ORDER BY (reference_hash = $2) DESC NULLS LAST, created_at, id LIMIT 1`, input.SourceApp, reference, fingerprint)
	item, err := scanCandidate(row, false)
	if err == nil {
		if reference != nil {
			_, err = db.Exec(ctx, `UPDATE finance.expense_candidates SET reference_hash = $2
				WHERE id = $1::uuid AND reference_hash IS NULL`, item.ID, reference)
		}
		return item, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return item, false, err
	}
	item, err = scanCandidate(db.QueryRow(ctx, `INSERT INTO finance.expense_candidates
		(source_app, reference_hash, fingerprint, amount_paise, merchant, occurred_on)
		VALUES ($1, $2, $3, $4, $5, $6::date) RETURNING `+candidateColumns,
		input.SourceApp, reference, fingerprint, input.AmountPaise, input.Merchant, input.OccurredOn), false)
	return item, err == nil, err
}

func (s *Store) listCandidates(ctx context.Context) ([]captureCandidate, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT `+candidateColumns+`,
		EXISTS (SELECT FROM finance.transactions t WHERE t.user_id = c.user_id AND t.kind = 'expense'
			AND t.voided_at IS NULL AND t.occurred_on = c.occurred_on AND t.amount_paise = c.amount_paise)
		OR EXISTS (SELECT FROM finance.expense_candidates other WHERE other.user_id = c.user_id AND other.status = 'pending'
			AND other.id <> c.id AND other.occurred_on = c.occurred_on AND other.amount_paise = c.amount_paise)
		FROM finance.expense_candidates c WHERE status = 'pending' ORDER BY created_at, id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []captureCandidate{}
	for rows.Next() {
		item, err := scanCandidate(rows, true)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) candidateByID(ctx context.Context, id uuid.UUID) (captureCandidate, error) {
	item, err := scanCandidate(database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT `+candidateColumns+` FROM finance.expense_candidates WHERE id = $1::uuid`, id.String()), false)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, errCandidateNotFound
	}
	return item, err
}

func (s *Store) expenseExists(ctx context.Context, input NewTransaction) (bool, error) {
	var exists bool
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT EXISTS (SELECT FROM finance.transactions
		WHERE kind = 'expense' AND voided_at IS NULL AND account_id = $1::uuid AND occurred_on = $2::date
		AND amount_paise = $3)`, input.AccountID.String(), input.OccurredOn, input.AmountPaise).Scan(&exists)
	return exists, err
}

func (s *Store) reviewCandidate(ctx context.Context, id uuid.UUID, status string, transaction *string, at time.Time) error {
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx, `UPDATE finance.expense_candidates
		SET status = $2, transaction_id = $3::uuid, reviewed_at = $4, merchant = ''
		WHERE id = $1::uuid AND status = 'pending'`, id.String(), status, transaction, at.UTC())
	if err == nil && tag.RowsAffected() == 0 {
		return errCandidateReviewed
	}
	return err
}

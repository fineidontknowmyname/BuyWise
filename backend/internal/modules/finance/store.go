package finance

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

const transactionColumns = `id::text, account_id::text, category_id::text, kind, amount_paise,
	occurred_on::text, note, idempotency_key, voided_at, void_reason, created_at`

const accountSummarySelect = `SELECT a.id::text, a.name, a.opening_balance_paise, a.created_at,
	a.opening_balance_paise + COALESCE(SUM(
		CASE t.kind WHEN 'income' THEN t.amount_paise WHEN 'expense' THEN -t.amount_paise END
	), 0)::bigint
FROM finance.accounts a
LEFT JOIN finance.transactions t ON t.account_id = a.id AND t.voided_at IS NULL`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

func (s *Store) CreateAccount(ctx context.Context, account Account) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO finance.accounts (id, name, opening_balance_paise, created_at) VALUES ($1::uuid, $2, $3, $4)`,
		account.ID.String(), account.Name, account.OpeningBalancePaise, account.CreatedAt)
	if isUniqueViolation(err) {
		return ErrDuplicateName
	}
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

func (s *Store) ListAccounts(ctx context.Context) ([]AccountSummary, error) {
	rows, err := s.pool.Query(ctx, accountSummarySelect+` GROUP BY a.id ORDER BY a.id`)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	list := []AccountSummary{}
	for rows.Next() {
		summary, err := scanAccountSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("list accounts: %w", err)
		}
		list = append(list, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return list, nil
}

func (s *Store) GetAccount(ctx context.Context, id uuid.UUID) (AccountSummary, error) {
	row := s.pool.QueryRow(ctx, accountSummarySelect+` WHERE a.id = $1::uuid GROUP BY a.id`, id.String())
	summary, err := scanAccountSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountSummary{}, ErrAccountNotFound
	}
	if err != nil {
		return AccountSummary{}, fmt.Errorf("get account: %w", err)
	}
	return summary, nil
}

func scanAccountSummary(row pgx.Row) (AccountSummary, error) {
	var summary AccountSummary
	var id string
	if err := row.Scan(&id, &summary.Name, &summary.OpeningBalancePaise, &summary.CreatedAt, &summary.BalancePaise); err != nil {
		return AccountSummary{}, err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return AccountSummary{}, fmt.Errorf("parse account id: %w", err)
	}
	summary.ID = parsed
	summary.CreatedAt = summary.CreatedAt.UTC()
	return summary, nil
}

func (s *Store) CreateCategory(ctx context.Context, category Category) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO finance.categories (id, name, kind, committed) VALUES ($1::uuid, $2, $3, $4)`,
		category.ID.String(), category.Name, string(category.Kind), category.Committed)
	if isUniqueViolation(err) {
		return ErrDuplicateName
	}
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, name, kind, committed FROM finance.categories ORDER BY kind DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	list := []Category{}
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("list categories: %w", err)
		}
		list = append(list, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return list, nil
}

func (s *Store) GetCategory(ctx context.Context, id uuid.UUID) (Category, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id::text, name, kind, committed FROM finance.categories WHERE id = $1::uuid`, id.String())
	category, err := scanCategory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrCategoryNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return category, nil
}

func scanCategory(row pgx.Row) (Category, error) {
	var category Category
	var id, kind string
	if err := row.Scan(&id, &category.Name, &kind, &category.Committed); err != nil {
		return Category{}, err
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return Category{}, fmt.Errorf("parse category id: %w", err)
	}
	category.ID = parsed
	category.Kind = Kind(kind)
	return category, nil
}

func (s *Store) CreateTransaction(ctx context.Context, tx Transaction) (Transaction, bool, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO finance.transactions
			(id, account_id, category_id, kind, amount_paise, occurred_on, note, idempotency_key, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::date, $7, $8, $9)
		ON CONFLICT (idempotency_key) DO NOTHING`,
		tx.ID.String(), tx.AccountID.String(), tx.CategoryID.String(), string(tx.Kind), tx.AmountPaise,
		tx.OccurredOn.Format(DateLayout), tx.Note, tx.IdempotencyKey, tx.CreatedAt)
	if err != nil {
		return Transaction{}, false, fmt.Errorf("create transaction: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return tx, true, nil
	}
	row := s.pool.QueryRow(ctx,
		`SELECT `+transactionColumns+` FROM finance.transactions WHERE idempotency_key = $1`, tx.IdempotencyKey)
	existing, err := scanTransaction(row)
	if err != nil {
		return Transaction{}, false, fmt.Errorf("load existing transaction: %w", err)
	}
	return existing, false, nil
}

func (s *Store) ListTransactions(ctx context.Context, accountID uuid.UUID) ([]Transaction, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+transactionColumns+` FROM finance.transactions
		WHERE account_id = $1::uuid ORDER BY occurred_on DESC, id DESC`, accountID.String())
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	list := []Transaction{}
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("list transactions: %w", err)
		}
		list = append(list, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return list, nil
}

func (s *Store) VoidTransaction(ctx context.Context, id uuid.UUID, at time.Time, reason string) (Transaction, error) {
	row := s.pool.QueryRow(ctx,
		`UPDATE finance.transactions SET voided_at = $2, void_reason = $3
		WHERE id = $1::uuid AND voided_at IS NULL RETURNING `+transactionColumns,
		id.String(), at, reason)
	tx, err := scanTransaction(row)
	if err == nil {
		return tx, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, fmt.Errorf("void transaction: %w", err)
	}
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM finance.transactions WHERE id = $1::uuid)`, id.String()).Scan(&exists); err != nil {
		return Transaction{}, fmt.Errorf("void transaction: %w", err)
	}
	if exists {
		return Transaction{}, ErrAlreadyVoided
	}
	return Transaction{}, ErrTransactionNotFound
}

func scanTransaction(row pgx.Row) (Transaction, error) {
	var tx Transaction
	var id, accountID, categoryID, kind, occurredOn string
	if err := row.Scan(&id, &accountID, &categoryID, &kind, &tx.AmountPaise, &occurredOn,
		&tx.Note, &tx.IdempotencyKey, &tx.VoidedAt, &tx.VoidReason, &tx.CreatedAt); err != nil {
		return Transaction{}, err
	}
	var err error
	if tx.ID, err = uuid.Parse(id); err != nil {
		return Transaction{}, fmt.Errorf("parse transaction id: %w", err)
	}
	if tx.AccountID, err = uuid.Parse(accountID); err != nil {
		return Transaction{}, fmt.Errorf("parse account id: %w", err)
	}
	if tx.CategoryID, err = uuid.Parse(categoryID); err != nil {
		return Transaction{}, fmt.Errorf("parse category id: %w", err)
	}
	if tx.OccurredOn, err = time.Parse(DateLayout, occurredOn); err != nil {
		return Transaction{}, fmt.Errorf("parse date: %w", err)
	}
	tx.Kind = Kind(kind)
	tx.CreatedAt = tx.CreatedAt.UTC()
	if tx.VoidedAt != nil {
		voided := tx.VoidedAt.UTC()
		tx.VoidedAt = &voided
	}
	return tx, nil
}

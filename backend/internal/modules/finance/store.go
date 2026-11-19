package finance

import (
	"context"
	"errors"
	"fmt"
	"selfemi/internal/platform/database"
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
	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
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
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, accountSummarySelect+` GROUP BY a.id ORDER BY a.id`)
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
	row := database.Scoped(ctx, s.pool).QueryRow(ctx, accountSummarySelect+` WHERE a.id = $1::uuid GROUP BY a.id`, id.String())
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
	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO finance.categories (id, name, kind, committed, archived) VALUES ($1::uuid, $2, $3, $4, $5)`,
		category.ID.String(), category.Name, string(category.Kind), category.Committed, category.Archived)
	if isUniqueViolation(err) {
		return ErrDuplicateName
	}
	if err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT id::text, name, kind, committed, COALESCE(archived, false) FROM finance.categories ORDER BY kind DESC, id`)
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
	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT id::text, name, kind, committed, COALESCE(archived, false) FROM finance.categories WHERE id = $1::uuid`, id.String())
	category, err := scanCategory(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrCategoryNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return category, nil
}

func (s *Store) UpdateCategory(ctx context.Context, category Category) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`UPDATE finance.categories SET name = $2, kind = $3, committed = $4, archived = $5 WHERE id = $1::uuid`,
		category.ID.String(), category.Name, string(category.Kind), category.Committed, category.Archived)
	if isUniqueViolation(err) {
		return ErrDuplicateName
	}
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

func (s *Store) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `DELETE FROM finance.categories WHERE id = $1::uuid`, id.String())
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}

func (s *Store) CategoryTransactionCount(ctx context.Context, id uuid.UUID) (int64, error) {
	var count int64
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT COUNT(*) FROM finance.transactions WHERE category_id = $1::uuid`, id.String()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count category transactions: %w", err)
	}
	return count, nil
}

func (s *Store) DeleteBudget(ctx context.Context, categoryID uuid.UUID) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `DELETE FROM finance.budgets WHERE category_id = $1::uuid`, categoryID.String())
	if err != nil {
		return fmt.Errorf("delete budget: %w", err)
	}
	return nil
}

func scanCategory(row pgx.Row) (Category, error) {
	var category Category
	var id, kind string
	if err := row.Scan(&id, &category.Name, &kind, &category.Committed, &category.Archived); err != nil {
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
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO finance.transactions
			(id, account_id, category_id, kind, amount_paise, occurred_on, note, idempotency_key, created_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::date, $7, $8, $9)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING`,
		tx.ID.String(), tx.AccountID.String(), tx.CategoryID.String(), string(tx.Kind), tx.AmountPaise,
		tx.OccurredOn.Format(DateLayout), tx.Note, tx.IdempotencyKey, tx.CreatedAt)
	if err != nil {
		return Transaction{}, false, fmt.Errorf("create transaction: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return tx, true, nil
	}
	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
		`SELECT `+transactionColumns+` FROM finance.transactions WHERE idempotency_key = $1`, tx.IdempotencyKey)
	existing, err := scanTransaction(row)
	if err != nil {
		return Transaction{}, false, fmt.Errorf("load existing transaction: %w", err)
	}
	return existing, false, nil
}

func (s *Store) ListTransactions(ctx context.Context, accountID uuid.UUID) ([]Transaction, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
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
	row := database.Scoped(ctx, s.pool).QueryRow(ctx,
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
	if err := database.Scoped(ctx, s.pool).QueryRow(ctx,
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

func (s *Store) UpsertBudget(ctx context.Context, budget Budget) error {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx,
		`INSERT INTO finance.budgets (category_id, limit_paise, updated_at) VALUES ($1::uuid, $2, $3)
		ON CONFLICT (category_id) DO UPDATE SET limit_paise = EXCLUDED.limit_paise, updated_at = EXCLUDED.updated_at`,
		budget.CategoryID.String(), budget.LimitPaise, budget.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save budget: %w", err)
	}
	return nil
}

func (s *Store) ListBudgets(ctx context.Context) ([]Budget, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT category_id::text, limit_paise, updated_at FROM finance.budgets ORDER BY category_id`)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	list := []Budget{}
	for rows.Next() {
		var budget Budget
		var id string
		if err := rows.Scan(&id, &budget.LimitPaise, &budget.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list budgets: %w", err)
		}
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("parse category id: %w", err)
		}
		budget.CategoryID = parsed
		budget.UpdatedAt = budget.UpdatedAt.UTC()
		list = append(list, budget)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	return list, nil
}

func (s *Store) SpentByCategory(ctx context.Context, fromInclusive, toExclusive time.Time) (map[uuid.UUID]int64, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx,
		`SELECT category_id::text, SUM(amount_paise)::bigint FROM finance.transactions
		WHERE kind = 'expense' AND voided_at IS NULL AND occurred_on >= $1::date AND occurred_on < $2::date
		GROUP BY category_id`,
		fromInclusive.Format(DateLayout), toExclusive.Format(DateLayout))
	if err != nil {
		return nil, fmt.Errorf("sum spending: %w", err)
	}
	defer rows.Close()

	spent := map[uuid.UUID]int64{}
	for rows.Next() {
		var id string
		var total int64
		if err := rows.Scan(&id, &total); err != nil {
			return nil, fmt.Errorf("sum spending: %w", err)
		}
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("parse category id: %w", err)
		}
		spent[parsed] = total
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sum spending: %w", err)
	}
	return spent, nil
}

package finance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"selfemi/internal/platform/database"
)

var errProfileConflict = errors.New("profile changed")
var errTransferConflict = errors.New("transfer key already used")
var errTransferNotFound = errors.New("transfer not found")

type profileItem struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Frequency   string `json:"frequency"`
	AmountPaise int64  `json:"amountPaise"`
}

type financialProfile struct {
	DebtStatus            string        `json:"debtStatus"`
	Revision              int           `json:"revision"`
	Items                 []profileItem `json:"items"`
	VariableIncomePercent int64         `json:"variableIncomePercent"`
	BufferPercent         int64         `json:"bufferPercent"`
	BufferMinimumPaise    int64         `json:"bufferMinimumPaise"`
	EmergencyTargetPaise  int64         `json:"emergencyTargetPaise"`
}

type capacity struct {
	Version              string `json:"version"`
	FixedIncomePaise     int64  `json:"fixedIncomePaise"`
	VariableIncomePaise  int64  `json:"variableIncomePaise"`
	CountedVariablePaise int64  `json:"countedVariablePaise"`
	CommittedPaise       int64  `json:"committedPaise"`
	DiscretionaryPaise   int64  `json:"discretionaryPaise"`
	BufferPaise          int64  `json:"bufferPaise"`
	SafeSavingsPaise     int64  `json:"safeSavingsPaise"`
}

func defaultProfile() financialProfile {
	return financialProfile{Items: []profileItem{}, VariableIncomePercent: 70, BufferPercent: 10, BufferMinimumPaise: 500000}
}

func (p *financialProfile) validate() bool {
	switch p.DebtStatus {
	case "", "current", "overdue":
	default:
		return false
	}
	if p.Revision < 0 || p.Revision >= 2147483647 || p.Items == nil || len(p.Items) > 100 ||
		p.VariableIncomePercent < 0 || p.VariableIncomePercent > 100 ||
		p.BufferPercent < 0 || p.BufferPercent > 100 ||
		p.BufferMinimumPaise < 0 || p.BufferMinimumPaise > MaxAmountPaise ||
		p.EmergencyTargetPaise < 0 || p.EmergencyTargetPaise > MaxAmountPaise {
		return false
	}
	for i := range p.Items {
		item := &p.Items[i]
		name, err := normalizeName(item.Name)
		if err != nil || item.AmountPaise <= 0 || item.AmountPaise > MaxAmountPaise {
			return false
		}
		item.Name = name
		switch item.Kind {
		case "fixed_income", "variable_income", "irregular_income", "fixed_expense", "essential_expense", "discretionary", "debt":
		default:
			return false
		}
		switch item.Frequency {
		case "weekly", "biweekly", "monthly", "annual":
			if item.Kind == "irregular_income" {
				return false
			}
		case "once":
			if item.Kind != "irregular_income" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (p financialProfile) estimate() capacity {
	result := capacity{Version: "monthly-v1"}
	for _, item := range p.Items {
		amount := item.AmountPaise
		switch item.Frequency {
		case "weekly":
			amount *= 52
		case "biweekly":
			amount *= 26
		case "monthly":
			amount *= 12
		case "once":
			continue
		}
		switch item.Kind {
		case "fixed_income":
			result.FixedIncomePaise += amount / 12
		case "variable_income":
			result.VariableIncomePaise += amount / 12
		case "discretionary":
			result.DiscretionaryPaise += (amount + 11) / 12
		default:
			result.CommittedPaise += (amount + 11) / 12
		}
	}
	result.CountedVariablePaise = result.VariableIncomePaise * p.VariableIncomePercent / 100
	result.BufferPaise = max(p.BufferMinimumPaise, (result.CommittedPaise*p.BufferPercent+99)/100)
	result.SafeSavingsPaise = max(0, result.FixedIncomePaise+result.CountedVariablePaise-result.CommittedPaise-result.DiscretionaryPaise-result.BufferPaise)
	return result
}

func (s *Store) loadProfile(ctx context.Context) (financialProfile, error) {
	result := defaultProfile()
	var raw []byte
	var revision int
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT revision, assumptions FROM finance.financial_profiles
        ORDER BY revision DESC LIMIT 1`).Scan(&revision, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(raw, &result)
	result.Revision = revision
	return result, err
}

func (s *Store) saveProfile(ctx context.Context, profile financialProfile) (financialProfile, error) {
	if err := database.LockPlanning(ctx, s.pool); err != nil {
		return profile, err
	}
	raw, err := json.Marshal(profile)
	if err != nil {
		return profile, err
	}
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO finance.financial_profiles (revision, assumptions)
        SELECT $1::integer + 1, $2::jsonb
        WHERE $1::integer = COALESCE((SELECT MAX(revision) FROM finance.financial_profiles), 0)
        ON CONFLICT (user_id, revision) DO NOTHING`, profile.Revision, raw)
	if err == nil && tag.RowsAffected() != 1 {
		return profile, errProfileConflict
	}
	profile.Revision++
	return profile, err
}

func (s *Store) designations(ctx context.Context) (map[string]string, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT account_id::text, purpose FROM finance.account_designations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var id, purpose string
		if err := rows.Scan(&id, &purpose); err != nil {
			return nil, err
		}
		result[id] = purpose
	}
	return result, rows.Err()
}

func (s *Store) designate(ctx context.Context, accountID uuid.UUID, purpose string) error {
	if err := database.LockPlanning(ctx, s.pool); err != nil {
		return err
	}
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO finance.account_designations (account_id, purpose)
        SELECT id, $2 FROM finance.accounts WHERE id = $1::uuid
        ON CONFLICT (user_id, account_id) DO UPDATE SET purpose = EXCLUDED.purpose`, accountID.String(), purpose)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return err
}

type transferInput struct {
	PurchaseManaged bool   `json:"-"`
	FromAccountID   string `json:"fromAccountId"`
	ToAccountID     string `json:"toAccountId"`
	AmountPaise     int64  `json:"amountPaise"`
	OccurredOn      string `json:"occurredOn"`
	Note            string `json:"note"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type transfer struct {
	transferInput
	Managed    bool    `json:"purchaseManaged,omitempty"`
	ID         string  `json:"id"`
	ReversesID *string `json:"reversesId"`
	Reversed   bool    `json:"reversed"`
}

func (i *transferInput) validate(today time.Time) bool {
	from, fromErr := uuid.Parse(i.FromAccountID)
	to, toErr := uuid.Parse(i.ToAccountID)
	date, dateErr := time.Parse(DateLayout, i.OccurredOn)
	i.Note = strings.TrimSpace(i.Note)
	if fromErr != nil || toErr != nil || from == to || dateErr != nil || date.After(today) ||
		i.AmountPaise <= 0 || i.AmountPaise > MaxAmountPaise || len([]rune(i.Note)) > MaxNoteLength ||
		!validKey(i.IdempotencyKey) || strings.HasPrefix(i.IdempotencyKey, "reverse-") {
		return false
	}
	i.FromAccountID, i.ToAccountID = from.String(), to.String()
	return true
}

const transferSelect = `SELECT t.id::text, t.from_account_id::text, t.to_account_id::text, t.amount_paise,
    to_char(t.occurred_on, 'YYYY-MM-DD'), t.note, t.idempotency_key, t.reverses_id::text,
    EXISTS (SELECT FROM finance.transfers r WHERE r.reverses_id = t.id), t.purchase_managed
    FROM finance.transfers t`

func scanTransfer(row pgx.Row) (transfer, error) {
	var entry transfer
	err := row.Scan(&entry.ID, &entry.FromAccountID, &entry.ToAccountID, &entry.AmountPaise,
		&entry.OccurredOn, &entry.Note, &entry.IdempotencyKey, &entry.ReversesID, &entry.Reversed, &entry.PurchaseManaged)
	entry.Managed = entry.PurchaseManaged
	return entry, err
}

func (s *Store) listTransfers(ctx context.Context) ([]transfer, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, transferSelect+` ORDER BY t.occurred_on DESC, t.id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []transfer{}
	for rows.Next() {
		item, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) createTransfer(ctx context.Context, input transferInput) (transfer, bool, error) {
	if err := database.LockPlanning(ctx, s.pool); err != nil {
		return transfer{}, false, err
	}
	db := database.Scoped(ctx, s.pool)
	tag, err := db.Exec(ctx, `INSERT INTO finance.transfers
		(from_account_id, to_account_id, amount_paise, occurred_on, note, idempotency_key, purchase_managed)
		SELECT a.id, b.id, $3, $4::date, $5, $6, $7 FROM finance.accounts a CROSS JOIN finance.accounts b
        WHERE a.id = $1::uuid AND b.id = $2::uuid
        ON CONFLICT (user_id, idempotency_key) DO NOTHING`,
		input.FromAccountID, input.ToAccountID, input.AmountPaise, input.OccurredOn, input.Note, input.IdempotencyKey, input.PurchaseManaged)
	if err != nil {
		return transfer{}, false, err
	}
	result, err := scanTransfer(db.QueryRow(ctx, transferSelect+` WHERE t.idempotency_key = $1`, input.IdempotencyKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return result, false, ErrAccountNotFound
	}
	if err == nil && result.transferInput != input {
		return result, false, errTransferConflict
	}
	return result, tag.RowsAffected() == 1, err
}

func (s *Store) reverseTransfer(ctx context.Context, id uuid.UUID, today time.Time) (transfer, error) {
	if err := database.LockPlanning(ctx, s.pool); err != nil {
		return transfer{}, err
	}
	db := database.Scoped(ctx, s.pool)
	_, err := db.Exec(ctx, `INSERT INTO finance.transfers
        (from_account_id, to_account_id, amount_paise, occurred_on, note, idempotency_key, reverses_id)
        SELECT to_account_id, from_account_id, amount_paise, $2::date, '', $3, id
		FROM finance.transfers WHERE id = $1::uuid AND reverses_id IS NULL AND NOT purchase_managed
        ON CONFLICT (user_id, reverses_id) DO NOTHING`,
		id.String(), today.Format(DateLayout), "reverse-"+id.String())
	if err != nil {
		return transfer{}, err
	}
	result, err := scanTransfer(db.QueryRow(ctx, transferSelect+` WHERE t.reverses_id = $1::uuid`, id.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return result, errTransferNotFound
	}
	return result, err
}

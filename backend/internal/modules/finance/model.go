package finance

import (
	"errors"
	"time"
	"uuid"
)

const (
	MaxNameLength      = 200
	MaxNoteLength      = 500
	MaxReasonLength    = 500
	MaxKeyLength       = 100
	MaxAmountPaise     = 1_000_000_000_000
	DateLayout         = "2006-01-02"
	businessZoneName   = "IST"
	businessZoneOffset = 5*60*60 + 30*60
)

var businessZone = time.FixedZone(businessZoneName, businessZoneOffset)

var (
	ErrInvalidName         = errors.New("name must be 1 to 200 characters")
	ErrInvalidAmount       = errors.New("amount must be a whole number of paise from 1 to 1,000,000,000,000")
	ErrInvalidOpening      = errors.New("opening balance must be within 1,000,000,000,000 paise")
	ErrInvalidDate         = errors.New("date must be a real calendar date as YYYY-MM-DD")
	ErrInvalidKind         = errors.New("kind must be income or expense")
	ErrInvalidNote         = errors.New("note must be at most 500 characters")
	ErrInvalidReason       = errors.New("reason must be at most 500 characters")
	ErrInvalidKey          = errors.New("idempotency key must be 1 to 100 letters, digits, dashes or underscores")
	ErrAccountNotFound     = errors.New("account not found")
	ErrCategoryNotFound    = errors.New("category not found")
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrKindMismatch        = errors.New("transaction kind does not match the category kind")
	ErrCommittedIncome     = errors.New("only expense categories can be committed")
	ErrDuplicateName       = errors.New("that name is already used")
	ErrAlreadyVoided       = errors.New("transaction is already void")
	ErrPurchaseManaged     = errors.New("this entry belongs to a recorded purchase")
	ErrKeyConflict         = errors.New("idempotency key was already used for a different transaction")
	ErrCategoryInUse       = errors.New("cannot delete category referenced by transactions; archive it instead")
)

type Kind string

const (
	Income  Kind = "income"
	Expense Kind = "expense"
)

func (k Kind) Valid() bool {
	return k == Income || k == Expense
}

type Account struct {
	ID                  uuid.UUID
	Name                string
	OpeningBalancePaise int64
	CreatedAt           time.Time
}

type AccountSummary struct {
	Account
	BalancePaise int64
}

type Category struct {
	ID        uuid.UUID
	Name      string
	Kind      Kind
	Committed bool
	Archived  bool
}

type Transaction struct {
	PurchaseManaged bool
	ID              uuid.UUID
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Kind            Kind
	AmountPaise     int64
	OccurredOn      time.Time
	Note            string
	IdempotencyKey  string
	VoidedAt        *time.Time
	VoidReason      string
	CreatedAt       time.Time
}

func (t Transaction) Voided() bool {
	return t.VoidedAt != nil
}

func (t Transaction) SameRequestAs(other Transaction) bool {
	return t.AccountID == other.AccountID &&
		t.CategoryID == other.CategoryID &&
		t.Kind == other.Kind &&
		t.AmountPaise == other.AmountPaise &&
		t.OccurredOn.Equal(other.OccurredOn) &&
		t.Note == other.Note
}

func SignedPaise(kind Kind, amountPaise int64) int64 {
	if kind == Expense {
		return -amountPaise
	}
	return amountPaise
}

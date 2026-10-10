package finance

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"
)

type Repository interface {
	BudgetRepository
	CreateAccount(ctx context.Context, account Account) error
	ListAccounts(ctx context.Context) ([]AccountSummary, error)
	GetAccount(ctx context.Context, id uuid.UUID) (AccountSummary, error)
	CreateCategory(ctx context.Context, category Category) error
	ListCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, id uuid.UUID) (Category, error)
	CreateTransaction(ctx context.Context, tx Transaction) (stored Transaction, created bool, err error)
	ListTransactions(ctx context.Context, accountID uuid.UUID) ([]Transaction, error)
	VoidTransaction(ctx context.Context, id uuid.UUID, at time.Time, reason string) (Transaction, error)
}

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() uuid.UUID
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now, newID: uuid.NewV7}
}

type NewTransaction struct {
	AccountID      uuid.UUID
	CategoryID     uuid.UUID
	Kind           Kind
	AmountPaise    int64
	OccurredOn     string
	Note           string
	IdempotencyKey string
}

func normalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > MaxNameLength {
		return "", ErrInvalidName
	}
	return trimmed, nil
}

func validKey(key string) bool {
	if key == "" || len(key) > MaxKeyLength {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func (s *Service) Today() time.Time {
	local := s.now().In(businessZone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) CreateAccount(ctx context.Context, name string, openingBalancePaise int64) (AccountSummary, error) {
	normalized, err := normalizeName(name)
	if err != nil {
		return AccountSummary{}, err
	}
	if openingBalancePaise < -MaxAmountPaise || openingBalancePaise > MaxAmountPaise {
		return AccountSummary{}, ErrInvalidOpening
	}
	account := Account{ID: s.newID(), Name: normalized, OpeningBalancePaise: openingBalancePaise, CreatedAt: s.now().UTC()}
	if err := s.repo.CreateAccount(ctx, account); err != nil {
		return AccountSummary{}, err
	}
	return AccountSummary{Account: account, BalancePaise: openingBalancePaise}, nil
}

func (s *Service) ListAccounts(ctx context.Context) ([]AccountSummary, error) {
	return s.repo.ListAccounts(ctx)
}

func (s *Service) GetAccount(ctx context.Context, id uuid.UUID) (AccountSummary, error) {
	return s.repo.GetAccount(ctx, id)
}

func (s *Service) CreateCategory(ctx context.Context, name string, kind Kind, committed bool) (Category, error) {
	normalized, err := normalizeName(name)
	if err != nil {
		return Category{}, err
	}
	if !kind.Valid() {
		return Category{}, ErrInvalidKind
	}
	if committed && kind != Expense {
		return Category{}, ErrCommittedIncome
	}
	category := Category{ID: s.newID(), Name: normalized, Kind: kind, Committed: committed}
	if err := s.repo.CreateCategory(ctx, category); err != nil {
		return Category{}, err
	}
	return category, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *Service) CreateTransaction(ctx context.Context, in NewTransaction) (tx Transaction, created bool, err error) {
	if !validKey(in.IdempotencyKey) {
		return Transaction{}, false, ErrInvalidKey
	}
	if !in.Kind.Valid() {
		return Transaction{}, false, ErrInvalidKind
	}
	if in.AmountPaise < 1 || in.AmountPaise > MaxAmountPaise {
		return Transaction{}, false, ErrInvalidAmount
	}
	if utf8.RuneCountInString(in.Note) > MaxNoteLength {
		return Transaction{}, false, ErrInvalidNote
	}
	occurredOn := s.Today()
	if in.OccurredOn != "" {
		parsed, err := time.Parse(DateLayout, in.OccurredOn)
		if err != nil {
			return Transaction{}, false, ErrInvalidDate
		}
		occurredOn = parsed
	}
	if _, err := s.repo.GetAccount(ctx, in.AccountID); err != nil {
		return Transaction{}, false, err
	}
	category, err := s.repo.GetCategory(ctx, in.CategoryID)
	if err != nil {
		return Transaction{}, false, err
	}
	if category.Kind != in.Kind {
		return Transaction{}, false, ErrKindMismatch
	}

	candidate := Transaction{
		ID:             s.newID(),
		AccountID:      in.AccountID,
		CategoryID:     in.CategoryID,
		Kind:           in.Kind,
		AmountPaise:    in.AmountPaise,
		OccurredOn:     occurredOn,
		Note:           strings.TrimSpace(in.Note),
		IdempotencyKey: in.IdempotencyKey,
		CreatedAt:      s.now().UTC(),
	}
	stored, created, err := s.repo.CreateTransaction(ctx, candidate)
	if err != nil {
		return Transaction{}, false, err
	}
	if !created && !stored.SameRequestAs(candidate) {
		return Transaction{}, false, ErrKeyConflict
	}
	return stored, created, nil
}

func (s *Service) ListTransactions(ctx context.Context, accountID uuid.UUID) ([]Transaction, error) {
	if _, err := s.repo.GetAccount(ctx, accountID); err != nil {
		return nil, err
	}
	return s.repo.ListTransactions(ctx, accountID)
}

func (s *Service) VoidTransaction(ctx context.Context, id uuid.UUID, reason string) (Transaction, error) {
	trimmed := strings.TrimSpace(reason)
	if utf8.RuneCountInString(trimmed) > MaxReasonLength {
		return Transaction{}, ErrInvalidReason
	}
	return s.repo.VoidTransaction(ctx, id, s.now().UTC(), trimmed)
}

package finance

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
	"uuid"
)

type memoryRepo struct {
	mu           sync.Mutex
	accounts     []Account
	categories   []Category
	transactions []Transaction
}

func newMemoryRepo() *memoryRepo { return &memoryRepo{} }

func (m *memoryRepo) balanceOf(account Account) int64 {
	total := account.OpeningBalancePaise
	for _, t := range m.transactions {
		if t.AccountID == account.ID && !t.Voided() {
			total += SignedPaise(t.Kind, t.AmountPaise)
		}
	}
	return total
}

func (m *memoryRepo) CreateAccount(_ context.Context, account Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.accounts {
		if strings.EqualFold(a.Name, account.Name) {
			return ErrDuplicateName
		}
	}
	m.accounts = append(m.accounts, account)
	return nil
}

func (m *memoryRepo) ListAccounts(context.Context) ([]AccountSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []AccountSummary{}
	for _, a := range m.accounts {
		out = append(out, AccountSummary{Account: a, BalancePaise: m.balanceOf(a)})
	}
	return out, nil
}

func (m *memoryRepo) GetAccount(_ context.Context, id uuid.UUID) (AccountSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.accounts {
		if a.ID == id {
			return AccountSummary{Account: a, BalancePaise: m.balanceOf(a)}, nil
		}
	}
	return AccountSummary{}, ErrAccountNotFound
}

func (m *memoryRepo) CreateCategory(_ context.Context, category Category) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.categories {
		if c.Kind == category.Kind && strings.EqualFold(c.Name, category.Name) {
			return ErrDuplicateName
		}
	}
	m.categories = append(m.categories, category)
	return nil
}

func (m *memoryRepo) ListCategories(context.Context) ([]Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Category{}, m.categories...), nil
}

func (m *memoryRepo) GetCategory(_ context.Context, id uuid.UUID) (Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.categories {
		if c.ID == id {
			return c, nil
		}
	}
	return Category{}, ErrCategoryNotFound
}

func (m *memoryRepo) CreateTransaction(_ context.Context, tx Transaction) (Transaction, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.transactions {
		if existing.IdempotencyKey == tx.IdempotencyKey {
			return existing, false, nil
		}
	}
	m.transactions = append(m.transactions, tx)
	return tx, true, nil
}

func (m *memoryRepo) ListTransactions(_ context.Context, accountID uuid.UUID) ([]Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Transaction{}
	for _, t := range m.transactions {
		if t.AccountID == accountID {
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].OccurredOn.After(out[j].OccurredOn) })
	return out, nil
}

func (m *memoryRepo) VoidTransaction(_ context.Context, id uuid.UUID, at time.Time, reason string) (Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.transactions {
		if t.ID != id {
			continue
		}
		if t.Voided() {
			return Transaction{}, ErrAlreadyVoided
		}
		t.VoidedAt = &at
		t.VoidReason = reason
		m.transactions[i] = t
		return t, nil
	}
	return Transaction{}, ErrTransactionNotFound
}

package finance

import (
	"context"
	"errors"
	"slices"
	"time"
	"uuid"
)

type PurchaseFinances struct {
	ReviewRequired  bool
	ProfileRevision int
	DebtStatus      string
	EmergencyTarget int64
	OperatingBuffer int64
	MonthlyMargin   int64
	Accounts        []PlanningAccount
}

func (s *Store) PurchaseDayBalances(ctx context.Context, day string) (map[string]int64, error) {
	accounts, err := s.accountSummaries(ctx, &day)
	if err != nil {
		return nil, err
	}
	balances := make(map[string]int64, len(accounts))
	for _, account := range accounts {
		balances[account.ID.String()] = account.BalancePaise
	}
	return balances, nil
}

func (s *Store) PostPurchaseExpense(ctx context.Context, account, category uuid.UUID, amount int64, day time.Time, note string, now time.Time) (string, error) {
	id := uuid.NewV7()
	entry, created, err := s.CreateTransaction(ctx, Transaction{ID: id, AccountID: account, CategoryID: category,
		Kind: Expense, AmountPaise: amount, OccurredOn: day, Note: note, CreatedAt: now.UTC(),
		IdempotencyKey: id.String(), PurchaseManaged: true})
	if err == nil && !created {
		err = errors.New("purchase ledger entry already exists")
	}
	return entry.ID.String(), err
}

func (s *Store) PostPurchaseTransfer(ctx context.Context, from, to string, amount int64, day, note string) (string, error) {
	entry, created, err := s.createTransfer(ctx, transferInput{FromAccountID: from, ToAccountID: to,
		AmountPaise: amount, OccurredOn: day, Note: note, IdempotencyKey: uuid.NewV7().String(), PurchaseManaged: true})
	if err == nil && !created {
		err = errors.New("purchase transfer already exists")
	}
	return entry.ID, err
}

func (s *Store) PurchaseFinances(ctx context.Context, now time.Time) (PurchaseFinances, error) {
	profile, err := s.loadProfile(ctx)
	if err != nil {
		return PurchaseFinances{}, err
	}
	day := now.In(businessZone).Format(DateLayout)
	accounts, err := s.accountSummaries(ctx, &day)
	if err != nil {
		return PurchaseFinances{}, err
	}
	purposes, err := s.designations(ctx)
	if err != nil {
		return PurchaseFinances{}, err
	}
	balances := make([]PlanningAccount, 0, len(accounts))
	for _, account := range accounts {
		id := account.ID.String()
		balances = append(balances, PlanningAccount{ID: id, Name: account.Name, BalancePaise: account.BalancePaise, Purpose: purposes[id]})
	}
	estimate, err := s.profileCapacity(ctx, profile)
	if err != nil {
		return PurchaseFinances{}, err
	}
	debt := profile.DebtStatus
	if profile.MoneyPlan != nil {
		_, _, protected, err := s.PlanningSnapshot(ctx)
		if err != nil {
			return PurchaseFinances{}, err
		}
		for i := range balances {
			for _, account := range protected {
				if balances[i].ID == account.ID {
					balances[i].ProtectedPaise = account.ProtectedPaise
				}
			}
		}
	}
	if debt == "" && !slices.ContainsFunc(profile.Items, func(item profileItem) bool { return item.Kind == "debt" }) {
		debt = "none_declared"
	}
	return PurchaseFinances{
		ReviewRequired:  profile.MoneyPlan != nil && profile.MoneyPlan.Period < monthKey(now),
		ProfileRevision: profile.Revision, DebtStatus: debt, EmergencyTarget: profile.EmergencyTargetPaise,
		OperatingBuffer: estimate.BufferPaise,
		MonthlyMargin:   estimate.FixedIncomePaise + estimate.CountedVariablePaise - estimate.CommittedPaise - estimate.DiscretionaryPaise,
		Accounts:        balances,
	}, nil
}

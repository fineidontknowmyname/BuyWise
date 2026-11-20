package finance

import (
	"context"
	"slices"
	"time"
)

type PurchaseFinances struct {
	ProfileRevision int
	DebtStatus      string
	EmergencyTarget int64
	OperatingBuffer int64
	MonthlyMargin   int64
	Accounts        []PlanningAccount
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
		balances = append(balances, PlanningAccount{id, account.Name, account.BalancePaise, purposes[id]})
	}
	estimate := profile.estimate()
	debt := profile.DebtStatus
	if debt == "" && !slices.ContainsFunc(profile.Items, func(item profileItem) bool { return item.Kind == "debt" }) {
		debt = "none_declared"
	}
	return PurchaseFinances{
		ProfileRevision: profile.Revision, DebtStatus: debt, EmergencyTarget: profile.EmergencyTargetPaise,
		OperatingBuffer: estimate.BufferPaise,
		MonthlyMargin:   estimate.FixedIncomePaise + estimate.CountedVariablePaise - estimate.CommittedPaise - estimate.DiscretionaryPaise,
		Accounts:        balances,
	}, nil
}

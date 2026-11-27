package app

import (
	"context"

	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
)

type planningFinance struct{ store *finance.Store }

func (p planningFinance) Snapshot(ctx context.Context) (int64, bool, []goals.PlanningAccount, error) {
	capacity, available, accounts, err := p.store.PlanningSnapshot(ctx)
	result := make([]goals.PlanningAccount, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, goals.PlanningAccount{
			ID: account.ID, Name: account.Name, BalancePaise: account.BalancePaise, Purpose: account.Purpose,
			ProtectedPaise: account.ProtectedPaise,
		})
	}
	return capacity, available, result, err
}

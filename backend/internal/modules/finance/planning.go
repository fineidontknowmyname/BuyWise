package finance

import (
	"context"
	"time"
)

type PlanningAccount struct {
	ID             string
	Name           string
	BalancePaise   int64
	Purpose        string
	ProtectedPaise int64
}

func (s *Store) PlanningSnapshot(ctx context.Context) (int64, bool, []PlanningAccount, error) {
	profile, err := s.loadProfile(ctx)
	if err != nil {
		return 0, false, nil, err
	}
	now := time.Now()
	if s.today != nil {
		now = s.today()
	}
	day := now.In(businessZone).Format(DateLayout)
	accounts, err := s.accountSummaries(ctx, &day)
	if err != nil {
		return 0, false, nil, err
	}
	purposes, err := s.designations(ctx)
	if err != nil {
		return 0, false, nil, err
	}
	result := make([]PlanningAccount, 0, len(accounts))
	for _, account := range accounts {
		id := account.ID.String()
		result = append(result, PlanningAccount{ID: id, Name: account.Name, BalancePaise: account.BalancePaise, Purpose: purposes[id]})
	}
	estimate, err := s.profileCapacity(ctx, profile)
	if err != nil {
		return 0, false, nil, err
	}
	if profile.MoneyPlan != nil {
		view, err := s.moneyOverview(ctx, profile, now)
		if err != nil {
			return 0, false, nil, err
		}
		reserved := map[string]int64{}
		if s.reservations != nil {
			reserved, err = s.reservations(ctx)
			if err != nil {
				return 0, false, nil, err
			}
		}
		left := view.ProtectedPaise
		for _, account := range result {
			if account.Purpose != "emergency_reserve" {
				left += max(0, -account.BalancePaise)
			}
		}
		for i := range result {
			if result[i].Purpose == "emergency_reserve" {
				continue
			}
			amount := min(left, max(0, result[i].BalancePaise-reserved[result[i].ID]))
			result[i].ProtectedPaise = amount
			left -= amount
		}
		for i := range result {
			if result[i].Purpose == "emergency_reserve" {
				continue
			}
			amount := min(left, max(0, result[i].BalancePaise-result[i].ProtectedPaise))
			result[i].ProtectedPaise += amount
			left -= amount
		}
	}
	return estimate.SafeSavingsPaise, profile.MoneyPlan != nil && profile.MoneyPlan.Period == monthKey(now), result, nil
}

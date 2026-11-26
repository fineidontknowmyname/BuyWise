package finance

import "context"

type PlanningAccount struct {
	ID           string
	Name         string
	BalancePaise int64
	Purpose      string
}

func (s *Store) PlanningSnapshot(ctx context.Context) (int64, bool, []PlanningAccount, error) {
	profile, err := s.loadProfile(ctx)
	if err != nil {
		return 0, false, nil, err
	}
	accounts, err := s.ListAccounts(ctx)
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
		result = append(result, PlanningAccount{id, account.Name, account.BalancePaise, purposes[id]})
	}
	estimate, err := s.profileCapacity(ctx, profile)
	if err != nil {
		return 0, false, nil, err
	}
	return estimate.SafeSavingsPaise, profile.Revision > 0, result, nil
}

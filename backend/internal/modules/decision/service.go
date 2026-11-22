package decision

import (
	"context"
	"time"
	"uuid"
)

type Service struct {
	goals  GoalLookup
	inputs InputProvider
	store  *Store
	now    func() time.Time
}

func NewService(goals GoalLookup, inputs InputProvider, store *Store, now func() time.Time) *Service {
	return &Service{goals: goals, inputs: inputs, store: store, now: now}
}

func (s *Service) Evaluate(ctx context.Context, goalID uuid.UUID, save bool) (Decision, error) {
	if save && s.store == nil {
		return Decision{}, ErrHistoryUnavailable
	}
	if s.store != nil {
		if err := s.store.lock(ctx); err != nil {
			return Decision{}, err
		}
	}
	goal, err := s.goals.Goal(ctx, goalID)
	if err != nil {
		return Decision{}, err
	}
	now := s.now()
	var evidence Evidence
	if s.inputs != nil {
		evidence, err = s.inputs.Inputs(ctx, goal, now)
		if err != nil {
			return Decision{}, err
		}
	}
	result := Evaluate(goal, evidence, now)
	if save {
		result.ID = uuid.NewV7().String()
		result.Trigger = "user_reassessment"
		if err := s.store.save(ctx, result); err != nil {
			return Decision{}, err
		}
	}
	return result, nil
}

func (s *Service) History(ctx context.Context, goalID uuid.UUID) ([]Decision, error) {
	if _, err := s.goals.Goal(ctx, goalID); err != nil {
		return nil, err
	}
	if s.store == nil {
		return []Decision{}, nil
	}
	return s.store.history(ctx, goalID)
}

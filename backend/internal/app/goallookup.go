package app

import (
	"context"
	"errors"
	"uuid"

	"selfemi/internal/modules/decision"
	"selfemi/internal/modules/goals"
)

type goalLookup struct {
	goals *goals.Service
}

func (l goalLookup) Goal(ctx context.Context, id uuid.UUID) (decision.GoalRef, error) {
	goal, err := l.goals.Get(ctx, id)
	if errors.Is(err, goals.ErrNotFound) {
		return decision.GoalRef{}, decision.ErrGoalNotFound
	}
	if err != nil {
		return decision.GoalRef{}, err
	}
	return decision.GoalRef{ID: goal.ID, Name: goal.Name}, nil
}

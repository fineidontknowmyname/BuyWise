package decision

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"selfemi/internal/platform/provenance"
)

type Service struct {
	goals     GoalLookup
	providers []InputProvider
	engine    Engine
	now       func() time.Time
}

func NewService(goals GoalLookup, engine Engine, now func() time.Time, providers ...InputProvider) *Service {
	return &Service{goals: goals, providers: providers, engine: engine, now: now}
}

func (s *Service) Evaluate(ctx context.Context, goalID uuid.UUID) (Decision, error) {
	goal, err := s.goals.Goal(ctx, goalID)
	if err != nil {
		return Decision{}, err
	}
	var inputs []provenance.Labelled
	for _, provider := range s.providers {
		values, err := provider.Inputs(ctx, goal.ID)
		if err != nil {
			return Decision{}, fmt.Errorf("collect inputs: %w", err)
		}
		inputs = append(inputs, values...)
	}
	return s.engine.Decide(goal, inputs, s.now()), nil
}

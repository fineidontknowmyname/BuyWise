package decision

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"selfemi/internal/platform/provenance"
)

type fakeGoals struct {
	goal GoalRef
	err  error
}

func (f fakeGoals) Goal(context.Context, uuid.UUID) (GoalRef, error) { return f.goal, f.err }

type fixedInputs struct {
	values []provenance.Labelled
	seen   *uuid.UUID
}

func (f fixedInputs) Inputs(_ context.Context, goalID uuid.UUID) ([]provenance.Labelled, error) {
	if f.seen != nil {
		*f.seen = goalID
	}
	return f.values, nil
}

type failingInputs struct{}

func (failingInputs) Inputs(context.Context, uuid.UUID) ([]provenance.Labelled, error) {
	return nil, errors.New("boom")
}

var now = func() time.Time { return time.Date(2026, 10, 9, 15, 30, 0, 0, time.FixedZone("IST", 19800)) }

func TestEvaluateCombinesInputsInOrder(t *testing.T) {
	goal := GoalRef{ID: uuid.NewV7(), Name: "Laptop"}
	var seen uuid.UUID
	first := fixedInputs{values: []provenance.Labelled{{Key: "a", Source: provenance.Stub}}, seen: &seen}
	second := fixedInputs{values: []provenance.Labelled{{Key: "b", Source: provenance.Manual}}}

	d, err := NewService(fakeGoals{goal: goal}, StubEngine{}, now, first, second).Evaluate(context.Background(), goal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if seen != goal.ID {
		t.Fatalf("provider got goal %v, want %v", seen, goal.ID)
	}
	if d.GoalID != goal.ID || d.State != Reassess || d.RulesVersion != StubRulesVersion {
		t.Fatalf("decision = %+v", d)
	}
	if len(d.Inputs) != 2 || d.Inputs[0].Key != "a" || d.Inputs[1].Key != "b" || d.AllInputsReal() {
		t.Fatalf("inputs = %+v", d.Inputs)
	}
	if d.DecidedAt.Location() != time.UTC || d.DecidedAt.Hour() != 10 {
		t.Fatalf("decidedAt = %v, want UTC 10:00", d.DecidedAt)
	}
}

func TestEvaluatePassesThroughGoalNotFound(t *testing.T) {
	svc := NewService(fakeGoals{err: ErrGoalNotFound}, StubEngine{}, now)
	if _, err := svc.Evaluate(context.Background(), uuid.NewV7()); !errors.Is(err, ErrGoalNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestEvaluateFailsWhenAnInputFails(t *testing.T) {
	svc := NewService(fakeGoals{goal: GoalRef{ID: uuid.NewV7()}}, StubEngine{}, now, failingInputs{})
	if _, err := svc.Evaluate(context.Background(), uuid.NewV7()); err == nil {
		t.Fatal("expected an error")
	}
}

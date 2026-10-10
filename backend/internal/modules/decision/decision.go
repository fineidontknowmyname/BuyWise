package decision

import (
	"context"
	"errors"
	"time"
	"uuid"

	"selfemi/internal/platform/provenance"
)

type State string

const (
	Buy      State = "BUY"
	Wait     State = "WAIT"
	SaveMore State = "SAVE_MORE"
	Reassess State = "REASSESS"
)

var ErrGoalNotFound = errors.New("goal not found")

type GoalRef struct {
	ID   uuid.UUID
	Name string
}

type Decision struct {
	GoalID       uuid.UUID
	GoalName     string
	State        State
	ReasonCodes  []string
	Inputs       []provenance.Labelled
	RulesVersion string
	DecidedAt    time.Time
}

func (d Decision) AllInputsReal() bool {
	return provenance.AllReal(d.Inputs)
}

type GoalLookup interface {
	Goal(ctx context.Context, id uuid.UUID) (GoalRef, error)
}

type InputProvider interface {
	Inputs(ctx context.Context, goalID uuid.UUID) ([]provenance.Labelled, error)
}

type Engine interface {
	Decide(goal GoalRef, inputs []provenance.Labelled, now time.Time) Decision
}

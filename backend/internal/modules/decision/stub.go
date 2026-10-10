package decision

import (
	"time"

	"selfemi/internal/platform/provenance"
)

const StubRulesVersion = "stub-v0"

type StubEngine struct{}

func (StubEngine) Decide(goal GoalRef, inputs []provenance.Labelled, now time.Time) Decision {
	return Decision{
		GoalID:       goal.ID,
		GoalName:     goal.Name,
		State:        Reassess,
		ReasonCodes:  []string{"stub_engine"},
		Inputs:       inputs,
		RulesVersion: StubRulesVersion,
		DecidedAt:    now.UTC(),
	}
}

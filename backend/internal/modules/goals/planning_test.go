package goals

import (
	"testing"
	"time"
	"uuid"
)

func TestReverseEMIProjection(t *testing.T) {
	today := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	deadline := "2026-07-31"
	item := planCandidate{
		Goal:        Goal{ID: uuid.NewV7(), Status: "active", TargetDate: &deadline, TargetPricePaise: 8000000},
		Terms:       planTerms{Mode: "intelligent", Frequency: "monthly", ApprovedPaise: 1000000},
		FundedPaise: 2000000,
	}
	result := project(item, 1000000, true, today)
	if result.State != "on_track" || result.EstimatedDate == nil || *result.EstimatedDate != deadline {
		t.Fatalf("unexpected six-month projection: %+v", result)
	}
	if date := projectDate(today, 1, "monthly"); date == nil || *date != "2026-02-28" {
		t.Fatalf("unexpected month-end projection: %v", date)
	}
	if got := project(item, 500000, true, today); got.State != "replan" || got.MonthlyAllocatedPaise != 500000 || got.RecommendedPaise != 500000 {
		t.Fatalf("unexpected capped plan: %+v", got)
	}
	if got := project(item, 0, true, today); got.EstimatedDate != nil || !got.OverCapacity {
		t.Fatalf("zero-capacity projection must be infeasible: %+v", got)
	}
	item.FundedPaise = 8000000
	if got := project(item, 0, true, today); got.State != "funded" || got.MonthlyAllocatedPaise != 0 {
		t.Fatalf("a funded goal must release recurring capacity: %+v", got)
	}
}

func TestReverseEMIPriorityAndFrequency(t *testing.T) {
	today := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	earlier, later := "2026-02-01", "2026-12-01"
	a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	candidates := []planCandidate{
		{Goal: Goal{ID: a, Priority: "low", TargetDate: &earlier}},
		{Goal: Goal{ID: b, Priority: "high", TargetDate: &later}},
		{Goal: Goal{ID: c, Priority: "high", TargetDate: &earlier}},
	}
	orderCandidates(candidates)
	if candidates[0].Goal.ID != c || candidates[1].Goal.ID != b || candidates[2].Goal.ID != a {
		t.Fatal("priority and deadlines were not respected")
	}
	item := planCandidate{
		Goal:  Goal{Status: "active", TargetPricePaise: 5200},
		Terms: planTerms{Mode: "simple", Frequency: "weekly", ApprovedPaise: 1000},
	}
	got := project(item, 3000, true, today)
	if got.MonthlyRequestedPaise != 4334 || got.EstimatedDate == nil ||
		*got.EstimatedDate != "2026-03-14" || got.State != "replan" {
		t.Fatalf("unexpected weekly plan: %+v", got)
	}
}

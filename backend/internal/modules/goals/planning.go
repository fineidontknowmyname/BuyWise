package goals

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"uuid"
)

const maxPlanningPaise int64 = 1000000000000

var errPlanConflict = errors.New("plan revision changed")
var errFundingConflict = errors.New("funding conflict")
var errPlanUnavailable = errors.New("goal cannot be changed in its current state")
var errContributionCapacity = errors.New("contribution exceeds allocated safe capacity")
var errPlanningAccount = errors.New("account is not available for goal funding")

type PlanningAccount struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BalancePaise int64  `json:"balancePaise"`
	Purpose      string `json:"purpose"`
}

type PlanningFinance interface {
	Snapshot(context.Context) (int64, bool, []PlanningAccount, error)
}

type planTerms struct {
	Revision      int    `json:"revision"`
	Mode          string `json:"mode"`
	Frequency     string `json:"frequency"`
	ApprovedPaise int64  `json:"approvedPaise"`
}

func defaultTerms() planTerms {
	return planTerms{Mode: "intelligent", Frequency: "monthly"}
}

func (p planTerms) valid() bool {
	return p.Revision >= 0 && p.Revision < 2147483647 && p.ApprovedPaise >= 0 && p.ApprovedPaise <= maxPlanningPaise &&
		(p.Mode == "simple" || p.Mode == "intelligent") &&
		(p.Frequency == "weekly" || p.Frequency == "biweekly" || p.Frequency == "monthly")
}

func periodsPerYear(frequency string) int64 {
	switch frequency {
	case "weekly":
		return 52
	case "biweekly":
		return 26
	default:
		return 12
	}
}

func projectDate(today time.Time, periods int64, frequency string) *string {
	if periods > 520 || periods < 0 {
		return nil
	}
	date := today
	switch frequency {
	case "weekly":
		date = today.AddDate(0, 0, int(periods)*7)
	case "biweekly":
		date = today.AddDate(0, 0, int(periods)*14)
	default:
		month := time.Date(today.Year(), today.Month()+time.Month(periods), 1, 0, 0, 0, 0, time.UTC)
		lastDay := month.AddDate(0, 1, -1).Day()
		date = month.AddDate(0, 0, min(today.Day(), lastDay)-1)
	}
	result := date.Format(time.DateOnly)
	return &result
}

type planCandidate struct {
	Goal        Goal
	Terms       planTerms
	FundedPaise int64
}

type goalProjection struct {
	RemainingPaise        int64   `json:"remainingPaise"`
	MonthlyRequestedPaise int64   `json:"monthlyRequestedPaise"`
	MonthlyAllocatedPaise int64   `json:"monthlyAllocatedPaise"`
	RecommendedPaise      int64   `json:"recommendedPaise"`
	EstimatedDate         *string `json:"estimatedDate"`
	State                 string  `json:"state"`
	OverCapacity          bool    `json:"overCapacity"`
}

func rank(priority string) int {
	switch priority {
	case "high":
		return 0
	case "low":
		return 2
	default:
		return 1
	}
}

func orderCandidates(items []planCandidate) {
	slices.SortFunc(items, func(a, b planCandidate) int {
		if difference := rank(a.Goal.Priority) - rank(b.Goal.Priority); difference != 0 {
			return difference
		}
		first, second := "9999-12-31", "9999-12-31"
		if a.Goal.TargetDate != nil {
			first = *a.Goal.TargetDate
		}
		if b.Goal.TargetDate != nil {
			second = *b.Goal.TargetDate
		}
		if difference := strings.Compare(first, second); difference != 0 {
			return difference
		}
		return strings.Compare(a.Goal.ID.String(), b.Goal.ID.String())
	})
}

func project(candidate planCandidate, available int64, hasProfile bool, today time.Time) goalProjection {
	goal, terms := candidate.Goal, candidate.Terms
	result := goalProjection{RemainingPaise: max(0, goal.TargetPricePaise-candidate.FundedPaise), State: goal.Status}
	if goal.Status != "active" {
		return result
	}
	if goal.TargetPricePaise <= 0 {
		result.State = "target_required"
		return result
	}
	if result.RemainingPaise == 0 {
		result.State = "funded"
		result.EstimatedDate = projectDate(today, 0, terms.Frequency)
		return result
	}
	periods := periodsPerYear(terms.Frequency)
	result.RecommendedPaise = min(maxPlanningPaise, result.RemainingPaise, available*12/periods)
	result.MonthlyRequestedPaise = (terms.ApprovedPaise*periods + 11) / 12
	result.MonthlyAllocatedPaise = min(available, result.MonthlyRequestedPaise)
	result.OverCapacity = result.MonthlyRequestedPaise > available
	if !hasProfile && terms.Mode == "intelligent" {
		result.State = "profile_required"
		return result
	}
	effective := terms.ApprovedPaise
	if terms.Mode == "intelligent" {
		effective = min(effective, result.MonthlyAllocatedPaise*12/periods)
	}
	if effective > 0 {
		count := 1 + (result.RemainingPaise-1)/effective
		result.EstimatedDate = projectDate(today, count, terms.Frequency)
	}
	result.State = "on_track"
	switch {
	case result.OverCapacity || (goal.TargetDate != nil && (result.EstimatedDate == nil || *result.EstimatedDate > *goal.TargetDate)):
		result.State = "replan"
	case effective == 0:
		result.State = "contribution_required"
	case result.EstimatedDate == nil:
		result.State = "replan"
	}
	return result
}

func planningDate(now time.Time) time.Time {
	local := now.In(time.FixedZone("IST", 19800))
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

type fundingEntry struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"accountId"`
	AmountPaise int64     `json:"amountPaise"`
	CreatedAt   time.Time `json:"createdAt"`
}

type fundingInput struct {
	AccountID      string `json:"accountId"`
	AmountPaise    int64  `json:"amountPaise"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (f *fundingInput) valid() bool {
	id, err := uuid.Parse(f.AccountID)
	if err != nil || f.AmountPaise <= 0 || f.AmountPaise > maxPlanningPaise || len(f.IdempotencyKey) < 1 || len(f.IdempotencyKey) > 100 {
		return false
	}
	for _, r := range f.IdempotencyKey {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	f.AccountID = id.String()
	return true
}

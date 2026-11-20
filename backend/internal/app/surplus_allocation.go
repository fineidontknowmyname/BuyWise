package app

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"
	"uuid"

	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
	"selfemi/internal/platform/httpjson"
)

const maxAllocationLines = 10

type allocationInput struct {
	AccountID   string `json:"accountId"`
	GoalID      string `json:"goalId"`
	AmountPaise int64  `json:"amountPaise"`
}

type allocationLine struct {
	allocationInput
	AccountName string        `json:"accountName"`
	GoalName    string        `json:"goalName,omitempty"`
	Purpose     string        `json:"purpose"`
	Sources     []namedAmount `json:"sources"`
}

type allocationEvent struct {
	Lines      []allocationLine `json:"lines"`
	RecordedAt *time.Time       `json:"recordedAt,omitempty"`
}

type lifecycleOption struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Purpose string `json:"purpose,omitempty"`
}

type lifecycleView struct {
	Purchase       *purchaseSummary  `json:"purchase"`
	RemainingPaise int64             `json:"remainingPaise"`
	Accounts       []lifecycleOption `json:"accounts"`
	Categories     []lifecycleOption `json:"categories"`
	Goals          []lifecycleOption `json:"goals"`
	Allocations    []allocationEvent `json:"allocations"`
}

func (p purchaseLifecycle) view(w http.ResponseWriter, r *http.Request, goal goals.Goal) error {
	ctx := r.Context()
	out := lifecycleView{Accounts: []lifecycleOption{}, Categories: []lifecycleOption{}, Goals: []lifecycleOption{}, Allocations: []allocationEvent{}}
	event, err := p.goals.Purchase(ctx, goal.ID)
	if err != nil {
		return err
	}
	out.Purchase, err = readPurchase(event)
	if err != nil {
		return err
	}
	if out.Purchase != nil {
		remaining, err := p.goals.RemainingSurplus(ctx, goal.ID)
		if err != nil {
			return err
		}
		for _, amount := range remaining {
			out.RemainingPaise += amount
		}
		events, err := p.goals.SurplusAllocations(ctx, goal.ID, "")
		if err != nil {
			return err
		}
		for _, event := range events {
			allocation, err := readAllocation(event)
			if err != nil {
				return err
			}
			out.Allocations = append(out.Allocations, allocation)
		}
		if out.RemainingPaise > 0 {
			items, err := p.goals.List(ctx)
			if err != nil {
				return err
			}
			for _, item := range items {
				if item.ID != goal.ID && item.Status == "active" && item.TargetPricePaise > item.AllocatedPaise {
					out.Goals = append(out.Goals, lifecycleOption{ID: item.ID.String(), Name: item.Name})
				}
			}
		}
	} else if goal.Status == "active" || goal.Status == "paused" {
		categories, err := p.finance.ListCategories(ctx)
		if err != nil {
			return err
		}
		for _, item := range categories {
			if item.Kind == finance.Expense && !item.Archived {
				out.Categories = append(out.Categories, lifecycleOption{ID: item.ID.String(), Name: item.Name})
			}
		}
	}
	if out.RemainingPaise > 0 || len(out.Categories) > 0 {
		finances, err := p.finance.PurchaseFinances(ctx, p.now())
		if err != nil {
			return err
		}
		for _, account := range finances.Accounts {
			out.Accounts = append(out.Accounts, lifecycleOption{account.ID, account.Name, account.Purpose})
		}
	}
	httpjson.Write(w, 200, out)
	return nil
}

func readAllocation(event goals.PurchaseEvent) (allocationEvent, error) {
	var allocation allocationEvent
	err := json.Unmarshal(event.Details, &allocation)
	allocation.RecordedAt = &event.CreatedAt
	return allocation, err
}

func (p purchaseLifecycle) allocate(w http.ResponseWriter, r *http.Request, goal goals.Goal) error {
	var request struct {
		Key       string            `json:"key"`
		Confirmed bool              `json:"confirmed"`
		Lines     []allocationInput `json:"lines"`
	}
	invalidAllocation := lifecycleProblem{"invalid_allocation", "Confirm between one and ten positive allocation amounts with valid destinations."}
	if httpjson.Decode(w, r, &request) != nil || !request.Confirmed || len(request.Lines) == 0 || len(request.Lines) > maxAllocationLines {
		return invalidAllocation
	}
	key, ok := normalizedID(request.Key)
	if !ok {
		return invalidAllocation
	}
	for i := range request.Lines {
		line := &request.Lines[i]
		account, ok := normalizedID(line.AccountID)
		if !ok || line.AmountPaise < 1 || line.AmountPaise > finance.MaxAmountPaise {
			return invalidAllocation
		}
		line.AccountID = account
		if line.GoalID != "" {
			id, ok := normalizedID(line.GoalID)
			if !ok || id == goal.ID.String() {
				return invalidAllocation
			}
			line.GoalID = id
		}
	}
	ctx := r.Context()
	prior, err := p.goals.SurplusAllocations(ctx, goal.ID, key)
	if err != nil {
		return err
	}
	if len(prior) > 0 {
		allocation, err := readAllocation(prior[0])
		if err != nil {
			return err
		}
		if !slices.EqualFunc(request.Lines, allocation.Lines, func(input allocationInput, saved allocationLine) bool {
			return input == saved.allocationInput
		}) {
			return requestChanged
		}
		httpjson.Write(w, 200, allocation)
		return nil
	}
	remaining, err := p.goals.RemainingSurplus(ctx, goal.ID)
	if err != nil {
		return err
	}
	now := p.now()
	finances, err := p.finance.PurchaseFinances(ctx, now)
	if err != nil {
		return err
	}
	reserves, err := p.goals.PurchaseReservations(ctx, goal.ID)
	if err != nil {
		return err
	}
	accounts := make(map[string]finance.PlanningAccount, len(finances.Accounts))
	backed := make(map[string]int64, len(remaining))
	for _, account := range finances.Accounts {
		accounts[account.ID] = account
		if account.Purpose != "emergency_reserve" {
			lot := remaining[account.ID]
			backed[account.ID] = max(0, min(lot, account.BalancePaise-(reserves[account.ID].OtherGoals-lot)))
		}
	}
	debited := make(map[string]int64)
	goalTotals := make(map[string]int64)
	allocation := allocationEvent{Lines: make([]allocationLine, 0, len(request.Lines))}
	for _, input := range request.Lines {
		account, exists := accounts[input.AccountID]
		if !exists {
			return lifecycleProblem{"invalid_destination", "Choose an account you own."}
		}
		line := allocationLine{allocationInput: input, AccountName: account.Name, Purpose: account.Purpose, Sources: []namedAmount{}}
		if input.GoalID == "" {
			if account.Purpose != "general_savings" && account.Purpose != "emergency_reserve" {
				return lifecycleProblem{"invalid_destination", "Choose a general savings or emergency reserve account."}
			}
		} else {
			if account.Purpose == "emergency_reserve" || account.BalancePaise < reserves[account.ID].OtherGoals {
				return lifecycleProblem{"invalid_destination", "Choose a funded, non-emergency account to hold the destination goal savings."}
			}
			id, _ := uuid.Parse(input.GoalID)
			destination, err := p.goals.Get(ctx, id)
			if err != nil {
				return err
			}
			goalTotals[input.GoalID] += input.AmountPaise
			if destination.Status != "active" || goalTotals[input.GoalID] > destination.TargetPricePaise-destination.AllocatedPaise {
				return lifecycleProblem{"invalid_destination", "Choose an active goal with enough room for this allocation."}
			}
			line.GoalName = destination.Name
		}
		amount := input.AmountPaise
		consume := func(source finance.PlanningAccount) {
			used := min(amount, backed[source.ID])
			if used > 0 {
				backed[source.ID] -= used
				debited[source.ID] += used
				amount -= used
				line.Sources = append(line.Sources, namedAmount{AccountID: source.ID, AccountName: source.Name, AmountPaise: used})
			}
		}
		consume(account)
		for _, source := range finances.Accounts {
			if amount == 0 {
				break
			}
			if source.ID != account.ID {
				consume(source)
			}
		}
		if amount > 0 {
			return lifecycleProblem{"surplus_unavailable", "The allocation exceeds available backed surplus. Check the source balances and remaining surplus."}
		}
		allocation.Lines = append(allocation.Lines, line)
	}
	for i := range allocation.Lines {
		line := &allocation.Lines[i]
		for j := range line.Sources {
			source := &line.Sources[j]
			if source.AccountID != line.AccountID {
				source.TransferID, err = p.finance.PostPurchaseTransfer(ctx, source.AccountID, line.AccountID,
					source.AmountPaise, indiaDay(now), "Purchase surplus: "+goal.Name)
				if err != nil {
					return err
				}
			}
		}
		if line.GoalID != "" {
			id, _ := uuid.Parse(line.GoalID)
			if err := p.goals.AddSurplusFunding(ctx, id, line.AccountID, line.AmountPaise); err != nil {
				return err
			}
		}
	}
	details, err := json.Marshal(allocation)
	if err != nil {
		return err
	}
	if err := p.goals.RecordSurplusAllocation(ctx, goal.ID, key, details, debited); err != nil {
		return err
	}
	events, err := p.goals.SurplusAllocations(ctx, goal.ID, key)
	if err == nil {
		allocation.RecordedAt = &events[0].CreatedAt
		httpjson.Write(w, 201, allocation)
	}
	return err
}

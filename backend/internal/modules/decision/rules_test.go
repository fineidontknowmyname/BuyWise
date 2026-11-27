package decision

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"
)

func paise(value int64) *int64 { return &value }

func decisionFixture() (GoalRef, Evidence, time.Time) {
	day := time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC)
	product := uuid.NewV7()
	goal := GoalRef{ID: uuid.NewV7(), Name: "Notebook", Status: "active", ProductID: &product, TargetPaise: 10000}
	offer := OfferEvidence{OfferID: "offer-a", ObservationID: "obs-a", ObservedOn: "2026-11-20", ObservedAt: &day,
		Availability: "in_stock", ItemPaise: paise(9000), TaxPaise: paise(0), DeliveryPaise: paise(1000), DiscountPaise: paise(0)}
	evidence := Evidence{
		Finances: FinancialEvidence{ProfileRevision: 1, DebtStatus: "current", MonthlyMarginPaise: 3000,
			EmergencyTargetPaise: 4000, OperatingBufferPaise: 2000,
			Accounts: []AccountEvidence{{BalancePaise: 21000, GoalReservedPaise: 15000, OtherReservedPaise: 4000},
				{Purpose: "emergency_reserve", BalancePaise: 4000, GoalReservedPaise: 4000}}},
		Prices: PriceEvidence{ProductID: product.String(), Offers: []OfferEvidence{offer}},
	}
	return goal, evidence, day
}

func TestDecisionMatrix(t *testing.T) {
	for _, test := range []struct {
		name              string
		ready, attractive bool
		want              State
	}{
		{"financially ready and attractive", true, true, Buy},
		{"financially ready and unattractive", true, false, Wait},
		{"financially not ready and attractive", false, true, SaveMore},
		{"financially not ready and unattractive", false, false, Reassess},
	} {
		t.Run(test.name, func(t *testing.T) {
			goal, evidence, now := decisionFixture()
			if !test.ready {
				evidence.Finances.Accounts[0].GoalReservedPaise = 5000
			}
			if !test.attractive {
				evidence.Prices.Offers[0].DeliveryPaise = paise(4000)
			}
			got := Evaluate(goal, evidence, now)
			if got.State != test.want || !reflect.DeepEqual(got, Evaluate(goal, evidence, now)) {
				t.Fatalf("expected stable verdict %s, got %+v", test.want, got)
			}
		})
	}
}

func TestDecisionQualifications(t *testing.T) {
	for _, test := range []struct {
		name, reason string
		want         State
		change       func(*GoalRef, *Evidence, *time.Time)
	}{
		{"different variant", "product_mismatch", Reassess, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Prices.ProductID = uuid.NewV7().String() }},
		{"exact freshness boundary", "observation_stale", Reassess, func(_ *GoalRef, _ *Evidence, now *time.Time) { *now = now.Add(48 * time.Hour) }},
		{"stock is outside purchase check", "within_target", Buy, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Prices.Offers[0].Availability = "out_of_stock" }},
		{"delivery is optional", "within_target", Buy, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Prices.Offers[0].DeliveryPaise = nil }},
		{"paused", "goal_inactive", Reassess, func(g *GoalRef, _ *Evidence, _ *time.Time) { g.Status = "paused" }},
		{"reserve below target", "emergency_shortfall", SaveMore, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.Accounts[1].BalancePaise-- }},
		{"emergency savings excluded", "goal_funds_shortfall", SaveMore, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.Accounts[0].GoalReservedPaise = 9000 }},
		{"other earmarks protected", "goal_funds_shortfall", SaveMore, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.Accounts[0].OtherReservedPaise = 13000 }},
		{"cash buffer protected", "cash_buffer_shortfall", SaveMore, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.Accounts[0].BalancePaise = 15000 }},
		{"monthly obligations protected", "monthly_buffer_shortfall", SaveMore, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.MonthlyMarginPaise = 1999 }},
		{"debt unknown", "debt_unconfirmed", Reassess, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.DebtStatus = "" }},
		{"debt overdue", "debt_overdue", SaveMore, func(_ *GoalRef, e *Evidence, _ *time.Time) { e.Finances.DebtStatus = "overdue" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			goal, evidence, now := decisionFixture()
			test.change(&goal, &evidence, &now)
			got := Evaluate(goal, evidence, now)
			if got.State != test.want || !slices.Contains(got.ReasonCodes, test.reason) {
				t.Fatalf("expected %s with %s, got %s with %v", test.want, test.reason, got.State, got.ReasonCodes)
			}
		})
	}
}

func TestHistoricalPriceAndMaximum(t *testing.T) {
	goal, evidence, now := decisionFixture()
	goal.TargetPaise = 0
	offer := &evidence.Prices.Offers[0]
	offer.HistoryIDs = []string{"1", "2", "3", "4"}
	offer.HistoryLowPaise = 9800
	offer.HistorySumPaise = 45000
	if got := Evaluate(goal, evidence, now); got.State != Reassess {
		t.Fatalf("insufficient history returned %s", got.State)
	}
	offer.HistoryIDs = append(offer.HistoryIDs, "5")
	offer.HistorySumPaise += 11000
	if got := Evaluate(goal, evidence, now); got.State != Buy {
		t.Fatalf("near historical low returned %s", got.State)
	}
	evidence.Prices.MaximumPaise = paise(9999)
	if got := Evaluate(goal, evidence, now); got.State != Wait {
		t.Fatalf("explicit maximum must override history, got %s", got.State)
	}
}

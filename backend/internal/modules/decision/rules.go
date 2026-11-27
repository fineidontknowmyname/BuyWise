package decision

import (
	"time"
)

var marketZone = time.FixedZone("IST", 19800)

const minimumHistoryDays = 5

var reasons = map[string]struct{ message, action string }{
	"funds_price_required":     {"Add a product price to compare it with your goal savings.", "Save a product price or record an item price on the offer."},
	"money_month_review":       {"Your spending plan needs a new-month review.", "Review unused spending in My Money before relying on a purchase recommendation."},
	"goal_inactive":            {"The goal is paused, completed, or cancelled.", "Resume an eligible goal before reassessing this purchase."},
	"profile_missing":          {"There is no saved financial profile.", "Save a financial profile with current income, expenses, debt and reserve settings."},
	"debt_unconfirmed":         {"The repayment status of recorded debt is unconfirmed.", "Confirm repayment status in Finances."},
	"debt_overdue":             {"The financial profile reports overdue debt.", "Address overdue debt and update the financial profile."},
	"emergency_shortfall":      {"The designated emergency balance is below the configured reserve target.", "Restore the emergency reserve."},
	"monthly_buffer_shortfall": {"Reliable monthly income after planned expenses cannot cover the operating buffer.", "Update current income and commitments or restore the monthly operating margin."},
	"goal_funds_shortfall":     {"Eligible account balances back less goal funding than the checkout total.", "Earmark or restore sufficient cash for this goal."},
	"cash_buffer_shortfall":    {"Paying this price would leave too little liquid cash after protecting other goals.", "Increase liquid cash or lower the purchase cost so the operating buffer remains intact."},
	"financial_ready":          {"Backed goal funds cover checkout, with emergency cash and the operating buffer protected.", ""},
	"product_missing":          {"No confirmed product is linked to this purchase goal.", "Link the exact product variant and its retailer offer."},
	"product_mismatch":         {"The price evidence does not belong to the linked product.", "Check the product and variant linked to this goal."},
	"offer_missing":            {"This product has no variant-confirmed retailer offer.", "Add a retailer offer and confirm the product variant."},
	"observation_missing":      {"There is no price observation for the confirmed offer.", "Record or refresh a price on the retailer offer."},
	"observation_stale":        {"The latest usable offer evidence is too old.", "Record a current quote or refresh the retailer offer."},
	"availability_unconfirmed": {"The latest observation does not confirm this offer is in stock.", "Confirm availability and record an updated offer."},
	"checkout_unknown":         {"No usable item price is saved for this offer.", "Save a product price or record an item price on the offer."},
	"above_maximum":            {"The checkout total exceeds the maximum purchase price.", "Wait for a checkout total within the maximum."},
	"within_target":            {"The assessed price meets your saved acceptable price.", ""},
	"historically_attractive":  {"The checkout total meets a historical discount benchmark for this same offer.", ""},
	"price_unattractive":       {"The checkout total misses the available price benchmarks.", "Wait for a qualifying price or review the saved thresholds."},
	"no_price_benchmark":       {"There is no acceptable price threshold or enough comparable history.", "Set an ideal or good price, or collect at least five comparable observation days."},
}

func Evaluate(goal GoalRef, evidence Evidence, now time.Time) Decision {
	d := Decision{GoalID: goal.ID.String(), GoalName: goal.Name, GoalStatus: goal.Status, GoalTargetPaise: goal.TargetPaise,
		State: Reassess, RulesVersion: RulesVersion, DecidedAt: now.UTC(), Trigger: "live_preview",
		Evidence: evidence, Reasons: []Reason{}, NextSteps: []string{},
		Assumptions: []string{
			"Financial readiness uses recorded liquid cash, designated reserves, the saved profile and declared debt status.",
			"Price readiness uses a recorded item price or saved product price. A complete recorded checkout breakdown is used when available; unknown extras are not assumed to be zero. Stock availability is not assessed. Forecasts are excluded.",
		}}
	d.Price, d.SelectedOffer, d.CheckoutPaise = assessPrice(goal, evidence.Prices, now)
	if d.CheckoutPaise != nil {
		d.PriceBasis = "saved_price"
		if d.SelectedOffer != nil {
			d.PriceBasis = "item_price"
			if d.SelectedOffer.completeCheckout() {
				d.PriceBasis = "checkout_total"
			}
		}
	}
	d.Finance = d.assessFinances()
	if goal.Status != "active" {
		d.ReasonCodes = append(d.ReasonCodes, "goal_inactive")
	} else if d.Finance.State != "UNKNOWN" && d.Price.State != "UNKNOWN" {
		switch {
		case d.Finance.State == "READY" && d.Price.State == "ATTRACTIVE":
			d.State = Buy
		case d.Finance.State == "READY":
			d.State = Wait
		case d.Price.State == "ATTRACTIVE":
			d.State = SaveMore
		}
	}
	d.ReasonCodes = append(d.ReasonCodes, d.Finance.ReasonCodes...)
	d.ReasonCodes = append(d.ReasonCodes, d.Price.ReasonCodes...)
	for _, code := range d.ReasonCodes {
		reason := reasons[code]
		d.Reasons = append(d.Reasons, Reason{Code: code, Message: reason.message})
		if reason.action != "" {
			d.NextSteps = append(d.NextSteps, reason.action)
		}
	}
	switch d.State {
	case Buy:
		d.Explanation = "The saved price meets your price criteria and goal savings cover it while preserving protected money."
		d.NextSteps = append(d.NextSteps, "Reassess if the seller, checkout total, availability or available funds change.")
	case Wait:
		d.Explanation = "Goal savings cover the assessed price, but it is above your price criteria."
	case SaveMore:
		d.Explanation = "The offer meets the price criteria, but the financial protections are not all met."
	default:
		d.Explanation = "Review the evidence and resolve the issues below before relying on a purchase recommendation."
	}
	return d
}

func (d *Decision) assessFinances() Axis {
	f := d.Evidence.Finances
	var availablePaise int64
	for _, account := range f.Accounts {
		if account.Purpose == "emergency_reserve" {
			d.EmergencyBalancePaise += account.BalancePaise
			continue
		}
		remaining := account.BalancePaise - account.OtherReservedPaise
		availablePaise += remaining
		d.BackedGoalPaise += max(0, min(account.GoalReservedPaise, remaining))
	}
	if d.CheckoutPaise != nil {
		after := availablePaise - *d.CheckoutPaise
		d.AfterPurchasePaise = &after
	}
	if f.ReviewRequired {
		return Axis{State: "UNKNOWN", ReasonCodes: []string{"money_month_review"}}
	}

	out := Axis{State: "UNKNOWN", ReasonCodes: []string{}}
	if d.CheckoutPaise == nil && f.ProfileRevision > 0 && f.DebtStatus != "" {
		out.ReasonCodes = append(out.ReasonCodes, "funds_price_required")
	}
	if f.ProfileRevision == 0 {
		out.ReasonCodes = append(out.ReasonCodes, "profile_missing")
	} else if f.DebtStatus == "" {
		out.ReasonCodes = append(out.ReasonCodes, "debt_unconfirmed")
	}
	if f.DebtStatus == "overdue" {
		out.ReasonCodes = append(out.ReasonCodes, "debt_overdue")
	}
	if f.ProfileRevision > 0 {
		if d.EmergencyBalancePaise < f.EmergencyTargetPaise {
			out.ReasonCodes = append(out.ReasonCodes, "emergency_shortfall")
		}
		if f.MonthlyMarginPaise < f.OperatingBufferPaise {
			out.ReasonCodes = append(out.ReasonCodes, "monthly_buffer_shortfall")
		}
	}
	if d.CheckoutPaise != nil {
		if d.BackedGoalPaise < *d.CheckoutPaise {
			out.ReasonCodes = append(out.ReasonCodes, "goal_funds_shortfall")
		}
		if *d.AfterPurchasePaise < f.OperatingBufferPaise {
			out.ReasonCodes = append(out.ReasonCodes, "cash_buffer_shortfall")
		}
	}
	if f.ProfileRevision == 0 || f.DebtStatus == "" || d.CheckoutPaise == nil {
		return out
	}
	if len(out.ReasonCodes) > 0 {
		out.State = "NOT_READY"
		return out
	}
	return Axis{State: "READY", ReasonCodes: []string{"financial_ready"}}
}

func assessPrice(goal GoalRef, evidence PriceEvidence, now time.Time) (Axis, *OfferEvidence, *int64) {
	if goal.ProductID == nil {
		return priceResult("UNKNOWN", "product_missing"), nil, nil
	}
	if evidence.ProductID != goal.ProductID.String() {
		return priceResult("UNKNOWN", "product_mismatch"), nil, nil
	}
	var chosen, fallback *OfferEvidence
	var checkout int64
	for i := range evidence.Offers {
		offer := &evidence.Offers[i]
		if fallback == nil || moreRecent(*offer, *fallback) {
			fallback = offer
		}
		total, problem := offer.checkout(now)
		if problem == "" && (chosen == nil || total < checkout || total == checkout && moreRecent(*offer, *chosen)) {
			chosen, checkout = offer, total
		}
	}
	if chosen == nil {
		if evidence.SavedPricePaise != nil && *evidence.SavedPricePaise > 0 && *evidence.SavedPricePaise <= 1000000000000 {
			return assessKnownPrice(goal, evidence, nil, *evidence.SavedPricePaise)
		}
		if fallback == nil {
			return priceResult("UNKNOWN", "offer_missing"), nil, nil
		}
		_, problem := fallback.checkout(now)
		return priceResult("UNKNOWN", problem), fallback, nil
	}
	return assessKnownPrice(goal, evidence, chosen, checkout)
}

func assessKnownPrice(goal GoalRef, evidence PriceEvidence, chosen *OfferEvidence, checkout int64) (Axis, *OfferEvidence, *int64) {

	maximum := goal.TargetPaise
	if evidence.MaximumPaise != nil {
		maximum = *evidence.MaximumPaise
	}
	acceptable := goal.TargetPaise
	if evidence.GoodPaise != nil {
		acceptable = *evidence.GoodPaise
	} else if evidence.IdealPaise != nil {
		acceptable = *evidence.IdealPaise
	}
	var days int64
	if chosen != nil && chosen.completeCheckout() {
		days = int64(len(chosen.HistoryIDs))
	}
	switch {
	case maximum > 0 && checkout > maximum:
		return priceResult("NOT_ATTRACTIVE", "above_maximum"), chosen, &checkout
	case acceptable > 0 && checkout <= acceptable:
		return priceResult("ATTRACTIVE", "within_target"), chosen, &checkout
	case days >= minimumHistoryDays &&
		(checkout*100 <= chosen.HistoryLowPaise*105 || checkout*days*100 <= chosen.HistorySumPaise*90):
		return priceResult("ATTRACTIVE", "historically_attractive"), chosen, &checkout
	case acceptable > 0 || days >= minimumHistoryDays:
		return priceResult("NOT_ATTRACTIVE", "price_unattractive"), chosen, &checkout
	default:
		return priceResult("UNKNOWN", "no_price_benchmark"), chosen, &checkout
	}
}

func priceResult(state, reason string) Axis {
	return Axis{State: state, ReasonCodes: []string{reason}}
}

func moreRecent(a, b OfferEvidence) bool {
	if a.ObservedOn != b.ObservedOn {
		return a.ObservedOn > b.ObservedOn
	}
	return a.ObservedAt != nil && (b.ObservedAt == nil || a.ObservedAt.After(*b.ObservedAt))
}

func (offer OfferEvidence) checkout(now time.Time) (int64, string) {
	if offer.ObservationID == "" {
		return 0, "observation_missing"
	}
	if offer.ObservedAt != nil {
		if offer.ObservedAt.After(now) || now.Sub(*offer.ObservedAt) >= 48*time.Hour {
			return 0, "observation_stale"
		}
	} else if day := now.In(marketZone).Format(time.DateOnly); offer.ObservedOn != day &&
		offer.ObservedOn != now.In(marketZone).AddDate(0, 0, -1).Format(time.DateOnly) {
		return 0, "observation_stale"
	}
	if offer.ItemPaise == nil {
		return 0, "checkout_unknown"
	}
	if !offer.completeCheckout() {
		return *offer.ItemPaise, ""
	}
	return *offer.ItemPaise + *offer.TaxPaise + *offer.DeliveryPaise - *offer.DiscountPaise, ""
}

func (offer OfferEvidence) completeCheckout() bool {
	return offer.ItemPaise != nil && offer.TaxPaise != nil && offer.DeliveryPaise != nil && offer.DiscountPaise != nil
}

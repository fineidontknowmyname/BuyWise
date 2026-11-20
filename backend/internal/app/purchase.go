package app

import (
	"context"
	"time"

	"selfemi/internal/modules/decision"
	"selfemi/internal/modules/finance"
	"selfemi/internal/modules/goals"
	"selfemi/internal/modules/pricing"
)

type purchaseInputs struct {
	finance *finance.Store
	goals   *goals.Store
	pricing *pricing.Store
}

func (p purchaseInputs) Inputs(ctx context.Context, goal decision.GoalRef, now time.Time) (decision.Evidence, error) {
	finances, err := p.finance.PurchaseFinances(ctx, now)
	if err != nil {
		return decision.Evidence{}, err
	}
	reservations, err := p.goals.PurchaseReservations(ctx, goal.ID)
	if err != nil {
		return decision.Evidence{}, err
	}
	accounts := make([]decision.AccountEvidence, 0, len(finances.Accounts))
	for _, account := range finances.Accounts {
		reserved := reservations[account.ID]
		accounts = append(accounts, decision.AccountEvidence{ID: account.ID, Purpose: account.Purpose,
			BalancePaise: account.BalancePaise, GoalReservedPaise: reserved.ForGoal, OtherReservedPaise: reserved.OtherGoals})
	}
	result := decision.Evidence{Finances: decision.FinancialEvidence{
		ProfileRevision: finances.ProfileRevision, DebtStatus: finances.DebtStatus, MonthlyMarginPaise: finances.MonthlyMargin,
		EmergencyTargetPaise: finances.EmergencyTarget, OperatingBufferPaise: finances.OperatingBuffer, Accounts: accounts,
	}, Prices: decision.PriceEvidence{Offers: []decision.OfferEvidence{}}}
	if goal.ProductID == nil {
		return result, nil
	}
	prices, err := p.pricing.PurchasePrices(ctx, *goal.ProductID, now)
	if err != nil {
		return decision.Evidence{}, err
	}
	result.Prices.ProductID, result.Prices.Variant = goal.ProductID.String(), prices.Variant
	result.Prices.IdealPaise, result.Prices.GoodPaise, result.Prices.MaximumPaise = prices.Ideal, prices.Good, prices.Maximum
	for _, price := range prices.Offers {
		result.Prices.Offers = append(result.Prices.Offers, decision.OfferEvidence{
			OfferID: price.OfferID, Retailer: price.Retailer, Seller: price.Seller, ObservationID: price.ObservationID,
			ObservedOn: price.ObservedOn, ObservedAt: price.ObservedAt, FetchedAt: price.FetchedAt,
			Source: price.Source, SourceLabel: price.SourceLabel, Availability: price.Availability,
			ItemPaise: price.ItemPaise, TaxPaise: price.TaxPaise, DeliveryPaise: price.DeliveryPaise,
			DiscountPaise: price.DiscountPaise, HistoryIDs: price.HistoryIDs, HistoryLowPaise: price.HistoryLow, HistorySumPaise: price.HistorySum,
		})
	}
	return result, nil
}

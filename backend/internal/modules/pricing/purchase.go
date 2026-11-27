package pricing

import (
	"context"
	"time"
	"uuid"

	"selfemi/internal/platform/database"
)

type PurchaseOffer struct {
	OfferID, Retailer, Seller, ObservationID, ObservedOn, Source, SourceLabel, Availability string
	ObservedAt, FetchedAt                                                                   *time.Time
	ItemPaise, TaxPaise, DeliveryPaise, DiscountPaise                                       *int64
	HistoryIDs                                                                              []string
	HistoryLow, HistorySum                                                                  int64
}

type PurchasePrices struct {
	SavedPrice           *int64
	Variant              string
	Ideal, Good, Maximum *int64
	Offers               []PurchaseOffer
}

func (s *Store) PurchasePrices(ctx context.Context, productID uuid.UUID, now time.Time) (PurchasePrices, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `
		SELECT p.variant, t.ideal_paise, t.good_paise, t.maximum_paise,
			CASE WHEN p.currency='INR' AND p.current_price_paise>0 THEN p.current_price_paise ELSE NULL END,
			COALESCE(o.id::text, ''), COALESCE(o.retailer, ''), COALESCE(o.seller, ''),
			COALESCE(latest.id::text, ''), COALESCE(to_char(latest.observed_on, 'YYYY-MM-DD'), ''),
			latest.observed_at, latest.fetched_at, COALESCE(latest.source, ''),
			COALESCE(latest.source_label, ''), COALESCE(latest.availability, ''),
			latest.item_paise, latest.tax_paise, latest.delivery_paise, latest.discount_paise,
			history.ids, COALESCE(history.low, 0), COALESCE(history.total, 0)
		FROM pricing.products p
		LEFT JOIN pricing.target_thresholds t ON t.product_id = p.id
		LEFT JOIN pricing.retailer_offers o ON o.product_id = p.id
		LEFT JOIN LATERAL (
			SELECT * FROM pricing.offer_observations r
			WHERE r.offer_id = o.id AND r.observed_on <= $2::date AND (r.observed_at IS NULL OR r.observed_at <= $3)
			ORDER BY r.observed_on DESC, r.observed_at DESC NULLS LAST, r.recorded_at DESC, r.id DESC LIMIT 1
		) latest ON true
		LEFT JOIN LATERAL (
			SELECT array_agg(id::text ORDER BY observed_on) AS ids, MIN(landed) AS low, SUM(landed)::bigint AS total
			FROM (
				SELECT DISTINCT ON (observed_on) id, observed_on, availability,
					item_paise + tax_paise + delivery_paise - discount_paise AS landed
				FROM pricing.offer_observations r
				WHERE r.offer_id = o.id AND r.observed_on >= $2::date - 90 AND r.observed_on < latest.observed_on
				ORDER BY observed_on, observed_at DESC NULLS LAST, recorded_at DESC, id DESC
			) daily WHERE landed IS NOT NULL
		) history ON true
		WHERE p.id = $1::uuid ORDER BY o.id`, productID.String(), marketDay(now), now)
	if err != nil {
		return PurchasePrices{}, err
	}
	defer rows.Close()
	result := PurchasePrices{}
	for rows.Next() {
		var offer PurchaseOffer
		if err := rows.Scan(&result.Variant, &result.Ideal, &result.Good, &result.Maximum,
			&result.SavedPrice,
			&offer.OfferID, &offer.Retailer, &offer.Seller, &offer.ObservationID, &offer.ObservedOn,
			&offer.ObservedAt, &offer.FetchedAt, &offer.Source, &offer.SourceLabel, &offer.Availability,
			&offer.ItemPaise, &offer.TaxPaise, &offer.DeliveryPaise, &offer.DiscountPaise,
			&offer.HistoryIDs, &offer.HistoryLow, &offer.HistorySum); err != nil {
			return PurchasePrices{}, err
		}
		if offer.OfferID != "" {
			result.Offers = append(result.Offers, offer)
		}
	}
	return result, rows.Err()
}

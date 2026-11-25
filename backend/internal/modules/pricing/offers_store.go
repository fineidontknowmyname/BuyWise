package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"selfemi/internal/platform/database"
)

func (s *Store) lockOfferProduct(ctx context.Context, id uuid.UUID) error {
	var marker int
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT 1 FROM pricing.products WHERE id = $1::uuid FOR UPDATE`, id.String()).Scan(&marker)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProductNotFound
	}
	return err
}

func (s *Store) createOffer(ctx context.Context, productID uuid.UUID, input offerInput) (retailerOffer, error) {
	if err := s.lockOfferProduct(ctx, productID); err != nil {
		return retailerOffer{}, err
	}
	row := database.Scoped(ctx, s.pool).QueryRow(ctx, `INSERT INTO pricing.retailer_offers
        (product_id, retailer, seller, listing_url)
        SELECT id, $2, $3, $4 FROM pricing.products
        WHERE id = $1::uuid AND btrim(variant) = $5
        ON CONFLICT DO NOTHING
        RETURNING id::text, product_id::text, retailer, seller, listing_url`,
		productID.String(), input.Retailer, input.Seller, input.ListingURL, strings.TrimSpace(input.Variant))
	offer, err := scanOffer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		product, err := s.GetProduct(ctx, productID)
		if err != nil {
			return retailerOffer{}, err
		}
		if strings.TrimSpace(input.Variant) != strings.TrimSpace(product.Variant) {
			return retailerOffer{}, errVariantMismatch
		}
		return retailerOffer{}, errOfferConflict
	}
	return offer, err
}

func scanOffer(row pgx.Row) (retailerOffer, error) {
	var offer retailerOffer
	err := row.Scan(&offer.ID, &offer.ProductID, &offer.Retailer, &offer.Seller, &offer.ListingURL)
	return offer, err
}

func (s *Store) listOffers(ctx context.Context, productID uuid.UUID) ([]retailerOffer, error) {
	rows, err := database.Scoped(ctx, s.pool).Query(ctx, `SELECT id::text, product_id::text, retailer, seller, listing_url
        FROM pricing.retailer_offers WHERE product_id = $1::uuid ORDER BY created_at, id`, productID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []retailerOffer{}
	for rows.Next() {
		offer, err := scanOffer(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, offer)
	}
	return result, rows.Err()
}

func (s *Store) getOffer(ctx context.Context, productID, offerID uuid.UUID) (retailerOffer, error) {
	row := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT id::text, product_id::text, retailer, seller, listing_url
        FROM pricing.retailer_offers WHERE product_id = $1::uuid AND id = $2::uuid`, productID.String(), offerID.String())
	offer, err := scanOffer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return offer, errOfferNotFound
	}
	return offer, err
}

func (s *Store) writeOfferPrices(ctx context.Context, offer retailerOffer, records []ObservedOfferPrice, source string, fetchedAt *time.Time) (int64, error) {
	encoded, err := json.Marshal(records)
	if err != nil {
		return 0, err
	}
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO pricing.offer_observations
        (offer_id, observed_on, observed_at, fetched_at, source, source_label, availability,
         item_paise, tax_paise, delivery_paise, discount_paise)
        SELECT $1::uuid, "observedOn", "observedAt", $2::timestamptz, $3, "sourceLabel", availability,
            "itemPaise", "taxPaise", "deliveryPaise", "discountPaise"
        FROM jsonb_to_recordset($4::jsonb) AS rows(
            "observedOn" date, "observedAt" timestamptz, "sourceLabel" text, availability text,
            "itemPaise" bigint, "taxPaise" bigint, "deliveryPaise" bigint, "discountPaise" bigint)
        ON CONFLICT DO NOTHING`, offer.ID, fetchedAt, source, encoded)
	return tag.RowsAffected(), err
}

func (s *Store) offerHistory(ctx context.Context, offer retailerOffer, now time.Time) (offerHistory, error) {
	db := database.Scoped(ctx, s.pool)
	result := offerHistory{Offer: offer, Freshness: "missing", Observations: []priceRecord{}}
	err := db.QueryRow(ctx, `SELECT COUNT(*), MIN(item_paise), MAX(item_paise), ROUND(AVG(item_paise))::bigint
        FROM pricing.offer_observations WHERE offer_id = $1::uuid`, offer.ID).Scan(
		&result.Stats.Count, &result.Stats.MinimumPaise, &result.Stats.MaximumPaise, &result.Stats.AveragePaise)
	if err != nil {
		return result, err
	}
	rows, err := db.Query(ctx, `SELECT id::text, to_char(observed_on, 'YYYY-MM-DD'), observed_at, fetched_at,
        recorded_at, source, source_label, availability, item_paise, tax_paise, delivery_paise, discount_paise
        FROM pricing.offer_observations WHERE offer_id = $1::uuid
        ORDER BY observed_on DESC, observed_at DESC NULLS LAST, recorded_at DESC, id DESC LIMIT $2`, offer.ID, maxHistoryRows)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var record priceRecord
		if err := rows.Scan(&record.ID, &record.ObservedOn, &record.ObservedAt, &record.FetchedAt, &record.RecordedAt,
			&record.Source, &record.SourceLabel, &record.Availability, &record.ItemPaise, &record.TaxPaise,
			&record.DeliveryPaise, &record.DiscountPaise); err != nil {
			return result, err
		}
		if record.TaxPaise != nil && record.DeliveryPaise != nil && record.DiscountPaise != nil {
			total := record.ItemPaise + *record.TaxPaise + *record.DeliveryPaise - *record.DiscountPaise
			record.LandedPaise = &total
		}
		result.Observations = append(result.Observations, record)
	}
	if len(result.Observations) > 0 {
		latest := result.Observations[0]
		result.Freshness = priceFreshness(latest.ObservedOn, now)
		if latest.ObservedAt != nil && now.Sub(*latest.ObservedAt) > priceFreshnessDays*24*time.Hour {
			result.Freshness = "stale"
		}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	rows.Close()
	result.Refresh, err = s.latestRefresh(ctx, offer.ID)
	return result, err
}

func (s *Store) getThresholds(ctx context.Context, productID uuid.UUID) (priceThresholds, error) {
	var result priceThresholds
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT ideal_paise, good_paise, maximum_paise
        FROM pricing.target_thresholds WHERE product_id = $1::uuid`, productID.String()).Scan(
		&result.IdealPaise, &result.GoodPaise, &result.MaximumPaise)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return result, err
}

func (s *Store) saveThresholds(ctx context.Context, productID uuid.UUID, thresholds priceThresholds) error {
	tag, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO pricing.target_thresholds
        (product_id, ideal_paise, good_paise, maximum_paise)
        SELECT id, $2, $3, $4 FROM pricing.products WHERE id = $1::uuid
        ON CONFLICT (user_id, product_id) DO UPDATE SET ideal_paise = EXCLUDED.ideal_paise,
            good_paise = EXCLUDED.good_paise, maximum_paise = EXCLUDED.maximum_paise`,
		productID.String(), thresholds.IdealPaise, thresholds.GoodPaise, thresholds.MaximumPaise)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return err
}

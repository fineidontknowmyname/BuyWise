package pricing

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

const serpPlanLimit = 250
const serpScheduledLimit = 220
const serpSource = "SerpApi · Amazon India"

type serpAccount struct {
	Renewal string `json:"plan_renewal_date"`
	Limit   int    `json:"searches_per_month"`
	Left    *int   `json:"plan_searches_left"`
	Used    *int   `json:"this_month_usage"`
}

func (s *SerpAPI) Observe(ctx context.Context, listing ProviderListing) (ObservedOfferPrice, error) {
	if !s.Supports(listing) {
		return ObservedOfferPrice{}, errProviderUnsupported
	}
	asin := amazonListingASIN(listing.URL)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ObservedOfferPrice{}, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_price_worker"); err != nil {
		return ObservedOfferPrice{}, err
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('buywise.serpapi', 0))"); err != nil {
		return ObservedOfferPrice{}, err
	}
	now := time.Now().UTC()
	var observed time.Time
	var paise *int64
	var availability *string
	err = tx.QueryRow(ctx, `SELECT observed_at, item_paise, availability FROM pricing.serpapi_observation_cache
        WHERE asin = $1 AND observed_at > $2`, asin, now.Add(-23*time.Hour)).
		Scan(&observed, &paise, &availability)
	if err == nil {
		if paise == nil {
			return ObservedOfferPrice{}, errProviderNoPrice
		}
		return ObservedOfferPrice{ObservedOn: marketDay(observed), ObservedAt: &observed, SourceLabel: serpSource,
			ItemPaise: *paise, Availability: *availability}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ObservedOfferPrice{}, err
	}
	if err := s.reserve(ctx, tx, listing.Scheduled); err != nil {
		return ObservedOfferPrice{}, err
	}
	price, lookupErr := s.fetchPrice(ctx, asin)
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if lookupErr == nil {
		observed, paise, availability = *price.ObservedAt, &price.ItemPaise, &price.Availability
	} else if errors.Is(lookupErr, errProviderNoPrice) {
		observed = now
	}
	if !observed.IsZero() {
		if _, err := tx.Exec(saveCtx, `INSERT INTO pricing.serpapi_observation_cache (asin, observed_at, item_paise, availability)
            VALUES ($1, $2, $3, $4) ON CONFLICT (asin) DO UPDATE SET observed_at = EXCLUDED.observed_at,
            item_paise = EXCLUDED.item_paise, availability = EXCLUDED.availability`, asin, observed, paise, availability); err != nil {
			return ObservedOfferPrice{}, err
		}
	}
	if err := tx.Commit(saveCtx); err != nil {
		return ObservedOfferPrice{}, err
	}
	return price, lookupErr
}

func (s *SerpAPI) reserve(ctx context.Context, tx pgx.Tx, scheduled bool) error {
	var account serpAccount
	if err := s.get(ctx, "/account.json", make(url.Values), &account); err != nil {
		return err
	}
	renewal, err := time.Parse(time.DateOnly, account.Renewal)
	if err != nil || account.Limit <= 0 || account.Left == nil || account.Used == nil || *account.Used < 0 {
		return ErrProviderUnavailable
	}
	limit := min(account.Limit, serpPlanLimit)
	if scheduled {
		limit = min(limit-30, serpScheduledLimit)
	}
	used := max(*account.Used, account.Limit-max(0, *account.Left))
	if *account.Left <= 0 || used >= limit {
		return errProviderQuota
	}
	tag, err := tx.Exec(ctx, `INSERT INTO pricing.serpapi_quota (renewal_on, reserved) VALUES ($1, $2 + 1)
        ON CONFLICT (renewal_on) DO UPDATE SET reserved = GREATEST(serpapi_quota.reserved, $2) + 1
        WHERE serpapi_quota.reserved < $3`, renewal, used, limit)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errProviderQuota
	}
	return nil
}

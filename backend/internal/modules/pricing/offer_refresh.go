package pricing

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"selfemi/internal/platform/database"
)

type refreshStatus struct {
	ID            string     `json:"id"`
	State         string     `json:"state"`
	Attempts      int        `json:"attempts"`
	ErrorCode     string     `json:"errorCode"`
	NextRefreshAt *time.Time `json:"nextRefreshAt"`
}

type refreshLease struct {
	ID        string
	UserID    string
	OfferID   string
	Token     string
	Attempts  int
	Scheduled bool
}

func (s *Store) queueRefresh(ctx context.Context, offer retailerOffer) (refreshStatus, error) {
	_, err := database.Scoped(ctx, s.pool).Exec(ctx, `INSERT INTO pricing.offer_refresh_jobs (offer_id)
        VALUES ($1::uuid) ON CONFLICT DO NOTHING`, offer.ID)
	if err != nil {
		return refreshStatus{}, err
	}
	result, err := s.latestRefresh(ctx, offer.ID)
	if err != nil {
		return refreshStatus{}, err
	}
	if result == nil {
		return refreshStatus{}, errOfferNotFound
	}
	return *result, nil
}

func (s *Store) latestRefresh(ctx context.Context, offerID string) (*refreshStatus, error) {
	var status refreshStatus
	err := database.Scoped(ctx, s.pool).QueryRow(ctx, `SELECT id::text, state, attempts, error_code, next_refresh_at
        FROM pricing.offer_refresh_jobs WHERE offer_id = $1::uuid
        ORDER BY created_at DESC, id DESC LIMIT 1`, offerID).Scan(
		&status.ID, &status.State, &status.Attempts, &status.ErrorCode, &status.NextRefreshAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &status, err
}

type RefreshWorker struct {
	store            *Store
	provider         PriceProvider
	now              func() time.Time
	upcomingPurchase func(context.Context, uuid.UUID, string) (bool, error)
}

func NewRefreshWorker(pool *pgxpool.Pool, provider PriceProvider, now func() time.Time) *RefreshWorker {
	return &RefreshWorker{store: NewStore(pool), provider: provider, now: now}
}

func (w *RefreshWorker) WithUpcomingPurchase(check func(context.Context, uuid.UUID, string) (bool, error)) *RefreshWorker {
	w.upcomingPurchase = check
	return w
}

func (w *RefreshWorker) Run(ctx context.Context, logger *slog.Logger) {
	if w.provider == nil {
		return
	}
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	var nextSchedule time.Time
	for {
		if _, ok := w.provider.(*SerpAPI); ok && !w.now().Before(nextSchedule) {
			nextSchedule = w.now().Add(time.Minute)
			if err := w.scheduleDue(ctx); err != nil && ctx.Err() == nil {
				logger.Error("price refresh scheduling failed", "error", err)
			}
		}
		if err := w.ProcessNext(ctx); err != nil && ctx.Err() == nil {
			logger.Error("price refresh job failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

func (w *RefreshWorker) scheduleDue(ctx context.Context) error {
	tx, err := w.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_price_worker"); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO pricing.offer_refresh_jobs (user_id, offer_id, scheduled)
        SELECT due.user_id, due.offer_id, true FROM pricing.offer_refresh_jobs due
        WHERE due.next_refresh_at <= now() AND NOT EXISTS (
            SELECT 1 FROM pricing.offer_refresh_jobs newer
            WHERE newer.user_id = due.user_id AND newer.offer_id = due.offer_id
              AND (newer.created_at, newer.id) > (due.created_at, due.id))
        ORDER BY due.next_refresh_at, due.id LIMIT 12 ON CONFLICT DO NOTHING`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *RefreshWorker) claim(ctx context.Context) (*refreshLease, error) {
	tx, err := w.store.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_price_worker"); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE pricing.offer_refresh_jobs SET state = 'failed', error_code = 'retry_limit',
        lease_until = NULL, lease_token = NULL WHERE state = 'running' AND attempts = 5 AND lease_until <= now()`); err != nil {
		return nil, err
	}
	var job refreshLease
	err = tx.QueryRow(ctx, `WITH eligible AS (
        SELECT id FROM pricing.offer_refresh_jobs
        WHERE attempts < 5 AND (state = 'queued' AND run_at <= now() OR state = 'running' AND lease_until <= now())
        ORDER BY scheduled, run_at, id FOR UPDATE SKIP LOCKED LIMIT 1
    )
    UPDATE pricing.offer_refresh_jobs job SET state = 'running', attempts = attempts + 1,
        lease_until = now() + interval '30 seconds', lease_token = uuidv7()
    FROM eligible WHERE job.id = eligible.id
    RETURNING job.id::text, job.user_id::text, job.offer_id::text, job.lease_token::text, job.attempts, job.scheduled`).
		Scan(&job.ID, &job.UserID, &job.OfferID, &job.Token, &job.Attempts, &job.Scheduled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}
	return &job, tx.Commit(ctx)
}

func scopeRefreshTenant(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_tenant"); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, "SELECT set_config('buywise.user_id', $1, true)", userID)
	return err
}

func (w *RefreshWorker) loadListing(ctx context.Context, job refreshLease) (retailerOffer, ProviderListing, error) {
	tx, err := w.store.pool.Begin(ctx)
	if err != nil {
		return retailerOffer{}, ProviderListing{}, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if err := scopeRefreshTenant(ctx, tx, job.UserID); err != nil {
		return retailerOffer{}, ProviderListing{}, err
	}
	offer, err := scanOffer(tx.QueryRow(ctx, `SELECT id::text, product_id::text, retailer, seller, listing_url
        FROM pricing.retailer_offers WHERE id = $1::uuid`, job.OfferID))
	if err != nil {
		return offer, ProviderListing{}, err
	}
	link, err := validateListing(offer.ListingURL)
	if err != nil {
		return offer, ProviderListing{}, err
	}
	id, err := uuid.Parse(offer.ProductID)
	if err != nil {
		return offer, ProviderListing{}, err
	}
	product, err := w.store.GetProduct(database.WithTenantTransaction(ctx, tx), id)
	if err != nil {
		return offer, ProviderListing{}, err
	}
	return offer, ProviderListing{URL: link.URL, Brand: product.Brand, Model: product.Model,
		Variant: product.Variant, Seller: offer.Seller, Scheduled: job.Scheduled}, nil
}

func (w *RefreshWorker) ProcessNext(ctx context.Context) error {
	if w.provider == nil {
		return ErrProviderUnavailable
	}
	job, err := w.claim(ctx)
	if err != nil || job == nil {
		return err
	}
	offer, listing, err := w.loadListing(ctx, *job)
	if err != nil {
		permanent := errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrProductNotFound) || errors.Is(err, errUnsupportedListing)
		return w.complete(ctx, *job, offer, nil, "listing_unavailable", !permanent)
	}
	providerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	price, providerErr := w.provider.Observe(providerCtx, listing)
	cancel()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.Is(providerErr, errProviderQuota):
		return w.complete(ctx, *job, offer, nil, "provider_quota_exhausted", false)
	case errors.Is(providerErr, errProviderNoPrice):
		return w.complete(ctx, *job, offer, nil, "provider_price_unconfirmed", false)
	case errors.Is(providerErr, errProviderUnsupported):
		return w.complete(ctx, *job, offer, nil, "provider_unsupported", false)
	case errors.Is(providerErr, errProviderCredentials):
		return w.complete(ctx, *job, offer, nil, "provider_credentials", false)
	case errors.Is(providerErr, ErrProviderRateLimited):
		return w.complete(ctx, *job, offer, nil, "provider_rate_limited", true)
	case providerErr != nil:
		return w.complete(ctx, *job, offer, nil, "provider_unavailable", true)
	case price.ObservedAt == nil || price.validate(w.now()) != nil || price.SourceLabel == "":
		return w.complete(ctx, *job, offer, nil, "invalid_provider_data", false)
	default:
		return w.complete(ctx, *job, offer, &price, "", false)
	}
}

func (w *RefreshWorker) complete(ctx context.Context, job refreshLease, offer retailerOffer, price *ObservedOfferPrice, code string, retry bool) error {
	tx, err := w.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_price_worker"); err != nil {
		return err
	}
	var id string
	err = tx.QueryRow(ctx, `SELECT id::text FROM pricing.offer_refresh_jobs
        WHERE id = $1::uuid AND state = 'running' AND lease_token = $2::uuid AND lease_until > now()
        FOR UPDATE`, job.ID, job.Token).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	state := "failed"
	var nextRefresh *time.Time
	if price != nil {
		if err := scopeRefreshTenant(ctx, tx, job.UserID); err != nil {
			return err
		}
		fetched := w.now().UTC()
		if _, err := w.store.writeOfferPrices(database.WithTenantTransaction(ctx, tx), offer, []ObservedOfferPrice{*price}, "provider", &fetched); err != nil {
			return err
		}
		if _, ok := w.provider.(*SerpAPI); ok {
			next, err := w.nextRefresh(ctx, tx, offer, *price)
			if err != nil {
				return err
			}
			nextRefresh = &next
		}
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE buywise_price_worker"); err != nil {
			return err
		}
		state = "succeeded"
	} else if retry && job.Attempts < 5 {
		state = "queued"
	}
	if state == "failed" && refreshSupported(w.provider, offer) {
		if _, ok := w.provider.(*SerpAPI); ok && code != "provider_credentials" {
			delay := 7 * 24 * time.Hour
			if code == "provider_quota_exhausted" {
				delay = 24 * time.Hour
			}
			next := w.now().Add(delay)
			nextRefresh = &next
		}
	}
	delaySeconds := 60 * (1 << (job.Attempts - 1))
	if _, err := tx.Exec(ctx, `UPDATE pricing.offer_refresh_jobs SET state = $2, error_code = $3,
        run_at = now() + make_interval(secs => $4), lease_until = NULL, lease_token = NULL, next_refresh_at = $5
        WHERE id = $1::uuid`, id, state, code, float64(delaySeconds), nextRefresh); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *RefreshWorker) nextRefresh(ctx context.Context, tx pgx.Tx, offer retailerOffer, price ObservedOfferPrice) (time.Time, error) {
	now := w.now()
	day := marketDay(now)
	var urgent bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (
        SELECT 1 FROM pricing.target_thresholds WHERE product_id = $1::uuid
        AND EXISTS (SELECT 1 FROM unnest(ARRAY[ideal_paise, good_paise, maximum_paise]) AS t(threshold)
            WHERE $2::bigint * 100 BETWEEN threshold * 90 AND threshold * 110)
    )`, offer.ProductID, price.ItemPaise).Scan(&urgent)
	if err == nil && !urgent && w.upcomingPurchase != nil {
		id, parseErr := uuid.Parse(offer.ProductID)
		if parseErr != nil {
			return time.Time{}, parseErr
		}
		urgent, err = w.upcomingPurchase(database.WithTenantTransaction(ctx, tx), id, day)
	}
	if urgent {
		return now.Add(24 * time.Hour), err
	}
	return now.Add(7 * 24 * time.Hour), err
}

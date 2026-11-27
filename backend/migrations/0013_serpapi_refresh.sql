ALTER TABLE pricing.offer_refresh_jobs ADD COLUMN scheduled boolean NOT NULL DEFAULT false;
ALTER TABLE pricing.offer_refresh_jobs ADD COLUMN next_refresh_at timestamptz;

CREATE INDEX offer_refresh_due ON pricing.offer_refresh_jobs (next_refresh_at)
    WHERE next_refresh_at IS NOT NULL;

GRANT INSERT (user_id, offer_id, scheduled), UPDATE (next_refresh_at)
    ON pricing.offer_refresh_jobs TO buywise_price_worker;

CREATE TABLE pricing.serpapi_quota (
    renewal_on date PRIMARY KEY,
    reserved integer NOT NULL CHECK (reserved BETWEEN 0 AND 250)
);

CREATE TABLE pricing.serpapi_observation_cache (
    asin text PRIMARY KEY CHECK (asin ~ '^[A-Z0-9]{10}$'),
    observed_at timestamptz NOT NULL,
    item_paise bigint CHECK (item_paise BETWEEN 1 AND 1000000000000),
    availability text CHECK (availability IN ('unknown', 'in_stock', 'out_of_stock', 'pre_order')),
    CHECK ((item_paise IS NULL) = (availability IS NULL))
);

ALTER TABLE pricing.serpapi_quota ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.serpapi_quota FORCE ROW LEVEL SECURITY;
ALTER TABLE pricing.serpapi_observation_cache ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.serpapi_observation_cache FORCE ROW LEVEL SECURITY;

CREATE POLICY serpapi_quota_worker ON pricing.serpapi_quota TO buywise_price_worker USING (true) WITH CHECK (true);
CREATE POLICY serpapi_cache_worker ON pricing.serpapi_observation_cache TO buywise_price_worker USING (true) WITH CHECK (true);

GRANT SELECT, INSERT, UPDATE ON pricing.serpapi_quota, pricing.serpapi_observation_cache TO buywise_price_worker;

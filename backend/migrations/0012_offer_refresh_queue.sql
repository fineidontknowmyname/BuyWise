DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'buywise_price_worker') THEN
        CREATE ROLE buywise_price_worker NOLOGIN NOSUPERUSER NOBYPASSRLS;
    END IF;
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'buywise_price_worker' AND (rolsuper OR rolbypassrls OR rolcanlogin)) THEN
        RAISE EXCEPTION 'buywise_price_worker must be a non-login role without RLS bypass';
    END IF;
END
$$;

CREATE TABLE pricing.offer_refresh_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    offer_id uuid NOT NULL,
    state text NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'running', 'succeeded', 'failed')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    run_at timestamptz NOT NULL DEFAULT now(),
    lease_until timestamptz,
    lease_token uuid,
    error_code text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, offer_id) REFERENCES pricing.retailer_offers (user_id, id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX offer_refresh_active ON pricing.offer_refresh_jobs (user_id, offer_id)
    WHERE state IN ('queued', 'running');
CREATE INDEX offer_refresh_claim ON pricing.offer_refresh_jobs (run_at, id) WHERE state IN ('queued', 'running');
CREATE INDEX offer_refresh_history ON pricing.offer_refresh_jobs (user_id, offer_id, created_at DESC, id DESC);

ALTER TABLE pricing.offer_refresh_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.offer_refresh_jobs FORCE ROW LEVEL SECURITY;
CREATE POLICY refresh_tenant_read ON pricing.offer_refresh_jobs FOR SELECT TO buywise_tenant
    USING (user_id = identity.current_user_id());
CREATE POLICY refresh_tenant_enqueue ON pricing.offer_refresh_jobs FOR INSERT TO buywise_tenant
    WITH CHECK (user_id = identity.current_user_id() AND state = 'queued' AND attempts = 0
        AND lease_until IS NULL AND lease_token IS NULL AND error_code = '');
CREATE POLICY refresh_worker ON pricing.offer_refresh_jobs TO buywise_price_worker
    USING (true) WITH CHECK (true);
GRANT SELECT, INSERT (offer_id) ON pricing.offer_refresh_jobs TO buywise_tenant;
GRANT USAGE ON SCHEMA pricing TO buywise_price_worker;
GRANT SELECT, UPDATE (state, attempts, run_at, lease_until, lease_token, error_code)
    ON pricing.offer_refresh_jobs TO buywise_price_worker;

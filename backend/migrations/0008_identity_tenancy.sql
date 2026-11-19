CREATE SCHEMA identity;
CREATE TABLE identity.users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    email text NOT NULL UNIQUE CHECK (char_length(email) BETWEEN 3 AND 254),
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE identity.sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id uuid NOT NULL REFERENCES identity.users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_user_id ON identity.sessions (user_id);
CREATE INDEX sessions_expires_at ON identity.sessions (expires_at);
CREATE TABLE identity.session_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES identity.users (id),
    event text NOT NULL CHECK (event IN ('signed_in', 'signed_out')),
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION identity.current_user_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('buywise.user_id', true), '')::uuid
$$;

ALTER TABLE finance.accounts
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE finance.accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.accounts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.accounts
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE finance.categories
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE finance.categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.categories FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.categories
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE finance.transactions
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE finance.transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.transactions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.transactions
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE finance.budgets
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, category_id);
ALTER TABLE finance.budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.budgets FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.budgets
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE goals.goals
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE goals.goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.goals FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.goals
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE pricing.products
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE pricing.products ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.products FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.products
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE pricing.price_import_batches
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE pricing.price_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.price_import_batches FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.price_import_batches
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE pricing.price_observations
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE pricing.price_observations ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.price_observations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.price_observations
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE pricing.document_uploads
    ADD COLUMN user_id uuid DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    ADD UNIQUE (user_id, id);
ALTER TABLE pricing.document_uploads ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.document_uploads FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.document_uploads
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE finance.transactions ADD CONSTRAINT tenant_reference_1
    FOREIGN KEY (user_id, account_id) REFERENCES finance.accounts (user_id, id);
ALTER TABLE finance.transactions ADD CONSTRAINT tenant_reference_2
    FOREIGN KEY (user_id, category_id) REFERENCES finance.categories (user_id, id);
ALTER TABLE finance.budgets ADD CONSTRAINT tenant_reference_3
    FOREIGN KEY (user_id, category_id) REFERENCES finance.categories (user_id, id);
ALTER TABLE goals.goals ADD CONSTRAINT tenant_reference_4
    FOREIGN KEY (user_id, product_id) REFERENCES pricing.products (user_id, id) ON DELETE SET NULL (product_id);
ALTER TABLE pricing.price_import_batches ADD CONSTRAINT tenant_reference_5
    FOREIGN KEY (user_id, product_id) REFERENCES pricing.products (user_id, id) ON DELETE CASCADE;
ALTER TABLE pricing.price_observations ADD CONSTRAINT tenant_reference_6
    FOREIGN KEY (user_id, product_id) REFERENCES pricing.products (user_id, id) ON DELETE CASCADE;
ALTER TABLE pricing.price_observations ADD CONSTRAINT tenant_reference_7
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.goals (user_id, id) ON DELETE SET NULL (goal_id);
ALTER TABLE pricing.price_observations ADD CONSTRAINT tenant_reference_8
    FOREIGN KEY (user_id, import_batch_id) REFERENCES pricing.price_import_batches (user_id, id) ON DELETE SET NULL (import_batch_id);
ALTER TABLE pricing.document_uploads ADD CONSTRAINT tenant_reference_9
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.goals (user_id, id) ON DELETE CASCADE;
ALTER TABLE pricing.document_uploads ADD CONSTRAINT tenant_reference_10
    FOREIGN KEY (user_id, product_id) REFERENCES pricing.products (user_id, id) ON DELETE SET NULL (product_id);

DROP INDEX finance.accounts_name_unique;
CREATE UNIQUE INDEX accounts_name_unique ON finance.accounts (user_id, lower(name)) NULLS NOT DISTINCT;
DROP INDEX finance.categories_kind_name_unique;
CREATE UNIQUE INDEX categories_kind_name_unique ON finance.categories (user_id, kind, lower(name)) NULLS NOT DISTINCT;
ALTER TABLE finance.transactions DROP CONSTRAINT transactions_idempotency_key_key;
ALTER TABLE finance.transactions ADD UNIQUE NULLS NOT DISTINCT (user_id, idempotency_key);

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'buywise_tenant') THEN
        CREATE ROLE buywise_tenant NOLOGIN NOSUPERUSER NOBYPASSRLS;
    END IF;
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'buywise_tenant' AND (rolsuper OR rolbypassrls OR rolcanlogin)) THEN
        RAISE EXCEPTION 'buywise_tenant must be a non-login role without RLS bypass';
    END IF;
END
$$;
GRANT USAGE ON SCHEMA identity, finance, goals, pricing TO buywise_tenant;
GRANT SELECT, INSERT, UPDATE, DELETE ON finance.accounts, finance.categories, finance.transactions, finance.budgets, goals.goals, pricing.products, pricing.price_import_batches, pricing.price_observations, pricing.document_uploads TO buywise_tenant;

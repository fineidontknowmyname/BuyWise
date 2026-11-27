CREATE TABLE finance.financial_profiles (
    user_id uuid NOT NULL DEFAULT identity.current_user_id() REFERENCES identity.users (id),
    revision integer NOT NULL CHECK (revision > 0),
    assumptions jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, revision)
);

CREATE TABLE finance.account_designations (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    account_id uuid NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('everyday', 'general_savings', 'emergency_reserve')),
    PRIMARY KEY (user_id, account_id),
    FOREIGN KEY (user_id, account_id) REFERENCES finance.accounts (user_id, id)
);

CREATE TABLE finance.transfers (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    from_account_id uuid NOT NULL,
    to_account_id uuid NOT NULL,
    amount_paise bigint NOT NULL CHECK (amount_paise BETWEEN 1 AND 1000000000000),
    occurred_on date NOT NULL,
    note text NOT NULL CHECK (char_length(note) <= 500),
    idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 100),
    reverses_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (from_account_id <> to_account_id),
    UNIQUE (user_id, id),
    UNIQUE (user_id, idempotency_key),
    UNIQUE (user_id, reverses_id),
    FOREIGN KEY (user_id, from_account_id) REFERENCES finance.accounts (user_id, id),
    FOREIGN KEY (user_id, to_account_id) REFERENCES finance.accounts (user_id, id),
    FOREIGN KEY (user_id, reverses_id) REFERENCES finance.transfers (user_id, id)
);
CREATE INDEX transfers_outgoing ON finance.transfers (user_id, from_account_id);
CREATE INDEX transfers_incoming ON finance.transfers (user_id, to_account_id);

ALTER TABLE finance.financial_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.financial_profiles FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.financial_profiles
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE finance.account_designations ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.account_designations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.account_designations
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE finance.transfers ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.transfers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.transfers
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

GRANT SELECT, INSERT ON finance.financial_profiles, finance.transfers TO buywise_tenant;
GRANT SELECT, INSERT, UPDATE ON finance.account_designations TO buywise_tenant;

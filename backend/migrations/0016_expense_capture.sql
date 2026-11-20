CREATE TABLE finance.expense_candidates (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    source_app text NOT NULL CHECK (char_length(source_app) BETWEEN 3 AND 160),
    reference_hash text CHECK (char_length(reference_hash) = 64),
    fingerprint text NOT NULL CHECK (char_length(fingerprint) = 64),
    amount_paise bigint NOT NULL CHECK (amount_paise BETWEEN 1 AND 1000000000000),
    merchant text NOT NULL DEFAULT '' CHECK (char_length(merchant) <= 120),
    occurred_on date NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'rejected')),
    transaction_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    reviewed_at timestamptz,
    UNIQUE (user_id, transaction_id),
    FOREIGN KEY (user_id, transaction_id) REFERENCES finance.transactions (user_id, id),
    CHECK ((status = 'pending') = (reviewed_at IS NULL)),
    CHECK ((status = 'confirmed') = (transaction_id IS NOT NULL))
);

CREATE UNIQUE INDEX expense_reference ON finance.expense_candidates (user_id, source_app, reference_hash)
    WHERE reference_hash IS NOT NULL;
CREATE INDEX expense_fingerprint ON finance.expense_candidates (user_id, fingerprint);
CREATE UNIQUE INDEX expense_unreferenced ON finance.expense_candidates (user_id, fingerprint)
    WHERE reference_hash IS NULL;
CREATE INDEX expense_pending ON finance.expense_candidates (user_id, created_at, id) WHERE status = 'pending';
CREATE INDEX expense_candidate_amount ON finance.expense_candidates (user_id, occurred_on, amount_paise) WHERE status = 'pending';
CREATE INDEX expense_transaction_amount ON finance.transactions (user_id, occurred_on, amount_paise) WHERE kind = 'expense' AND voided_at IS NULL;

ALTER TABLE finance.expense_candidates ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance.expense_candidates FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON finance.expense_candidates
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());
GRANT SELECT, INSERT ON finance.expense_candidates TO buywise_tenant;
GRANT UPDATE (status, transaction_id, reviewed_at, merchant, reference_hash) ON finance.expense_candidates TO buywise_tenant;

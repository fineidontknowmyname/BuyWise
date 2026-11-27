CREATE TABLE goals.plan_revisions (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    revision integer NOT NULL CHECK (revision > 0),
    mode text NOT NULL CHECK (mode IN ('simple', 'intelligent')),
    frequency text NOT NULL CHECK (frequency IN ('weekly', 'biweekly', 'monthly')),
    approved_paise bigint NOT NULL CHECK (approved_paise BETWEEN 0 AND 1000000000000),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, goal_id, revision),
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.goals (user_id, id)
);

CREATE TABLE goals.funding (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    account_id uuid NOT NULL,
    amount_paise bigint NOT NULL CHECK (amount_paise BETWEEN 1 AND 1000000000000),
    idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 100),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, idempotency_key),
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.goals (user_id, id),
    FOREIGN KEY (user_id, account_id) REFERENCES finance.accounts (user_id, id)
);
CREATE INDEX goal_funding_goal ON goals.funding (user_id, goal_id);
CREATE INDEX goal_funding_account ON goals.funding (user_id, account_id);

ALTER TABLE goals.plan_revisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.plan_revisions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.plan_revisions
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE goals.funding ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.funding FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.funding
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

GRANT SELECT, INSERT ON goals.plan_revisions, goals.funding TO buywise_tenant;

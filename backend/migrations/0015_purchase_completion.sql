CREATE TABLE goals.purchases (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    request_key uuid NOT NULL,
    transaction_id uuid NOT NULL,
    details jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, goal_id),
    UNIQUE (user_id, transaction_id),
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.goals (user_id, id),
    FOREIGN KEY (user_id, transaction_id) REFERENCES finance.transactions (user_id, id)
);

CREATE TABLE goals.purchase_surplus (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    account_id uuid NOT NULL,
    amount_paise bigint NOT NULL CHECK (amount_paise > 0),
    PRIMARY KEY (user_id, goal_id, account_id),
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.purchases (user_id, goal_id),
    FOREIGN KEY (user_id, account_id) REFERENCES finance.accounts (user_id, id)
);

CREATE TABLE goals.surplus_allocations (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    request_key uuid NOT NULL,
    details jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, goal_id, request_key),
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.purchases (user_id, goal_id)
);

CREATE TABLE goals.surplus_debits (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    request_key uuid NOT NULL,
    account_id uuid NOT NULL,
    amount_paise bigint NOT NULL CHECK (amount_paise > 0),
    PRIMARY KEY (user_id, goal_id, account_id, request_key),
    FOREIGN KEY (user_id, goal_id, request_key) REFERENCES goals.surplus_allocations (user_id, goal_id, request_key),
    FOREIGN KEY (user_id, goal_id, account_id) REFERENCES goals.purchase_surplus (user_id, goal_id, account_id)
);

ALTER TABLE goals.purchases ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.purchases FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.purchases USING (user_id = identity.current_user_id()) WITH CHECK (user_id = identity.current_user_id());
ALTER TABLE goals.purchase_surplus ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.purchase_surplus FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.purchase_surplus USING (user_id = identity.current_user_id()) WITH CHECK (user_id = identity.current_user_id());
ALTER TABLE goals.surplus_allocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.surplus_allocations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.surplus_allocations USING (user_id = identity.current_user_id()) WITH CHECK (user_id = identity.current_user_id());
ALTER TABLE goals.surplus_debits ENABLE ROW LEVEL SECURITY;
ALTER TABLE goals.surplus_debits FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON goals.surplus_debits USING (user_id = identity.current_user_id()) WITH CHECK (user_id = identity.current_user_id());

CREATE VIEW goals.remaining_surplus WITH (security_invoker = true) AS
SELECT s.user_id, s.goal_id, s.account_id, (s.amount_paise - COALESCE(SUM(d.amount_paise), 0))::bigint AS amount_paise
FROM goals.purchase_surplus s LEFT JOIN goals.surplus_debits d USING (user_id, goal_id, account_id)
GROUP BY s.user_id, s.goal_id, s.account_id
HAVING s.amount_paise > COALESCE(SUM(d.amount_paise), 0);

CREATE VIEW goals.reserved_funds WITH (security_invoker = true) AS
SELECT f.goal_id, f.account_id, f.amount_paise, false AS is_surplus
FROM goals.funding f JOIN goals.goals g ON g.user_id = f.user_id AND g.id = f.goal_id
WHERE g.status IN ('active', 'paused')
UNION ALL
SELECT goal_id, account_id, amount_paise, true FROM goals.remaining_surplus;

GRANT SELECT, INSERT ON goals.purchases, goals.purchase_surplus, goals.surplus_allocations, goals.surplus_debits TO buywise_tenant;
GRANT SELECT ON goals.remaining_surplus, goals.reserved_funds TO buywise_tenant;

ALTER TABLE finance.transactions ADD COLUMN purchase_managed boolean NOT NULL DEFAULT false;
ALTER TABLE finance.transfers ADD COLUMN purchase_managed boolean NOT NULL DEFAULT false;

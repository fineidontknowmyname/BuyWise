CREATE SCHEMA decision;

CREATE TABLE decision.purchase_decisions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    goal_id uuid NOT NULL,
    decided_at timestamptz NOT NULL,
    rules_version text NOT NULL,
    state text NOT NULL CHECK (state IN ('BUY', 'WAIT', 'SAVE_MORE', 'REASSESS')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    FOREIGN KEY (user_id, goal_id) REFERENCES goals.goals (user_id, id) ON DELETE CASCADE
);

CREATE INDEX purchase_decision_history ON decision.purchase_decisions (user_id, goal_id, decided_at DESC, id DESC);

ALTER TABLE decision.purchase_decisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE decision.purchase_decisions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON decision.purchase_decisions TO buywise_tenant
    USING (user_id = identity.current_user_id()) WITH CHECK (user_id = identity.current_user_id());

GRANT USAGE ON SCHEMA decision TO buywise_tenant;
GRANT SELECT, INSERT (id, goal_id, decided_at, rules_version, state, payload)
    ON decision.purchase_decisions TO buywise_tenant;

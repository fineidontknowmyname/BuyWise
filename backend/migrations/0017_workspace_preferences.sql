CREATE TABLE identity.preferences (
    user_id uuid PRIMARY KEY DEFAULT identity.current_user_id() REFERENCES identity.users(id) ON DELETE CASCADE,
    display_name text NOT NULL DEFAULT '' CHECK (char_length(display_name) <= 100),
    price_updates boolean NOT NULL DEFAULT true,
    reduced_motion boolean NOT NULL DEFAULT false
);

ALTER TABLE identity.preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity.preferences FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON identity.preferences
    USING (user_id = identity.current_user_id()) WITH CHECK (user_id = identity.current_user_id());
GRANT SELECT, INSERT, UPDATE (display_name, price_updates, reduced_motion) ON identity.preferences TO buywise_tenant;

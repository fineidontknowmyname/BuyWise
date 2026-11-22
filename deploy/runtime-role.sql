\getenv runtime_password BUYWISE_RUNTIME_PASSWORD

BEGIN;
SELECT pg_advisory_xact_lock(58320131617011);

SELECT 'CREATE ROLE buywise_runtime LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'buywise_runtime')
\gexec

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'buywise_runtime'
        AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolinherit OR NOT rolcanlogin)) THEN
        RAISE EXCEPTION 'buywise_runtime must be a restricted login role';
    END IF;
END
$$;

ALTER ROLE buywise_runtime PASSWORD :'runtime_password';
GRANT buywise_tenant, buywise_price_worker TO buywise_runtime WITH INHERIT FALSE;
GRANT USAGE ON SCHEMA identity TO buywise_runtime;
GRANT SELECT, INSERT ON identity.users TO buywise_runtime;
GRANT SELECT, INSERT, DELETE ON identity.sessions TO buywise_runtime;
GRANT INSERT ON identity.session_events TO buywise_runtime;

COMMIT;

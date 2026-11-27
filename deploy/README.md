# BuyWise deployment runbook

Use the [release rehearsal](REHEARSAL.md) after staging is reachable to seed synthetic workspaces, verify release journeys and collect sign-off evidence.

## Prepare staging

Use independent staging and production PostgreSQL 18 clusters, origins, provider keys, and backups. The public entry point is SvelteKit over HTTPS. Run Go and PostgreSQL on a private network. Android should use the same public origin as the browser. Its `/api/mobile` requests reach Go through the SvelteKit gateway.

Use Render Singapore for the first hosted staging rehearsal, subject to the project budget and confirmation that the database role setup works on the chosen plan. It supports the Go service and its in-process worker on a private network beside managed PostgreSQL. Check the recovery window and calculate the cost of running both environments before provisioning. [Render regions](https://render.com/docs/regions) · [Render pricing](https://render.com/pricing)

| Hosting option | Provisioning |
| --- | --- |
| Render | Budget for two service instances, database compute, and storage. Use a web service for Node, a private service for Go, and paid Render PostgreSQL. Use the internal database address and check point-in-time recovery. [Private networking](https://render.com/docs/private-network) · [Database recovery](https://render.com/docs/postgresql-backups) |
| Railway | Budget for a plan minimum and usage across CPU, memory, storage, and egress. Singapore is also available. Use separate Node, Go, and PostgreSQL services. Enable backups or point-in-time recovery and choose an availability level before setting recovery objectives. [Pricing](https://docs.railway.com/pricing) · [Regions](https://docs.railway.com/deployments/regions) · [PostgreSQL](https://docs.railway.com/databases/postgresql) |
| Single host | Use the included Compose setup. Manage host patching, encrypted off-host backups, and recovery yourself. The database volume shares the host's failure domain. |

1. Run the GitHub **Release gates** workflow and keep the release commit SHA. Require its jobs in the GitHub branch rules for the release branch. Rehearse from a clean database and a restored staging database before promoting the same release to production.
2. Provision PostgreSQL 18. Run `./migrate` from the API image with `DATABASE_URL` set to the migration or database administrator DSN. Use a direct database connection since migration locking is session-scoped. Then set `BUYWISE_RUNTIME_PASSWORD` in the same administrator environment and run `psql "$DATABASE_URL" -X -v ON_ERROR_STOP=1 -f runtime-role.sql`. This creates or updates the restricted API login and grants the two application roles. These roles are cluster-wide. The role setup can be replayed during recovery or password rotation. Avoid printing DSNs or the role password in deployment logs.
3. Start the API image normally with `API_ADDR=0.0.0.0:8080`, the restricted `DATABASE_URL`, and an optional `SERPAPI_API_KEY`. Point the platform's private health check at `/readyz`. TLS should be enabled for a managed database connection and verified against the provider CA where available. Use `sslmode=disable` only on the isolated network in the Compose example.
4. Start the web image with `API_URL` set to the private Go address, `ORIGIN` set to the exact HTTPS public origin, and an optional `GROQ_API_KEY`. Set `HOST=0.0.0.0` and the platform's `PORT` if deploying without the image. Use `/health` to check both app services and the database. Keep `DATABASE_URL`, SerpApi credentials, and migration credentials out of the web environment.
5. Confirm sign-up and sign-in, browser session cookie flags, signed-in page caching, HTTPS redirects, and the mobile gateway. Web writes expect the configured origin. The native client uses bearer authentication and requires no browser CORS policy. Verify that an unsigned session cannot fetch private data and that `/metrics` is unreachable on the public origin. Use two staging accounts to verify finance, goals, offers, and AI references stay scoped to the signed-in user.

For a single host, point the deployment hostname at the host, allow incoming TCP ports 80 and 443, copy `deploy/.env.example` to `deploy/.env`, and provide distinct random administrator and runtime passwords. URL-encode their DSN components. Set `BUYWISE_TAG` to the release commit SHA. Run from the repository root.

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml build migrate web
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d --wait --wait-timeout 120
curl --fail --silent --show-error https://YOUR_HOSTNAME/health
```

Keep secrets in the host or provider secret store. For rotation, update the stored value, run the privileged role setup for database credentials, recreate dependent services, verify readiness and sign-in, then revoke the old provider key. For a database password change, schedule a short maintenance window so pooled connections can drain and reconnect. Rolling back application images requires a schema-compatible release. Prefer a forward fix after a schema change. For data recovery, restore to a separate database and verify it before switching traffic.

If a migration fails, stop the rollout and compare the reported version with `schema_migrations`. Each migration commits its SQL and history record together. Diagnose the failure against staging with the same PostgreSQL version and role grants. Retry after correcting the cause, or apply a forward migration if an earlier version already committed. Keep the old app serving only while its schema is compatible with the versions that committed.

## Backup and recovery

Enable provider point-in-time recovery and schedule encrypted off-host logical exports. Start with a daily logical export and define the recovery point and recovery time objectives before launch. Treat dumps as sensitive financial data. Use `PGSERVICE` or an equivalent secret-backed connection instead of putting passwords into commands or logs. With PostgreSQL 18 client tools and separate source and empty restore databases, run the following from a trusted network.

```sh
PGSERVICE=buywise-source pg_dump --format=custom --no-owner --file=buywise.dump
PGSERVICE=buywise-restore pg_restore --exit-on-error --no-owner buywise.dump
```

Before restoring across clusters, provision the `buywise_tenant` and `buywise_price_worker` roles as `NOLOGIN` roles without RLS bypass. Restore with a role permitted to apply the stored grants and own the restored application objects. Run `runtime-role.sql` after restoring and point only the recovery instance at the new DSN. Compare `schema_migrations`, sample account and ledger balances, and pending refresh jobs. Use a synthetic account to verify sign-in, tenant isolation, the dashboard, and duplicate expense retries. Record backup age, restore duration, and where verification evidence is stored. Do not switch production traffic during a rehearsal.

## Worker and service monitoring

Scrape `GET /metrics` on the private Go service. The API exports process-local request and 5xx counters and the number of active requests. Go logs JSON request summaries with status, duration, and a generated `X-Request-ID`. Use proxy or hosting metrics for public route latency and reliability, including SvelteKit's assistant. Start with alerts for three consecutive failed readiness probes, a sustained 5xx rate over 1% on meaningful traffic, missed backup jobs, and a price refresh queue whose oldest runnable job has waited over 15 minutes. Adjust thresholds against staging measurements.

The price refresh worker runs inside Go when `SERPAPI_API_KEY` is set. To rehearse, enqueue a refresh in staging and restart the API during processing. Confirm the worker reclaims the expired lease or processes the queued job, and confirm the offer has only one resulting observation. Review `pricing.offer_refresh_jobs` using an authorized operator connection. Inspect `state`, `attempts`, `run_at`, `lease_until`, and `error_code`. For provider quota or credential failures, fix the provider first, then enqueue another refresh through the UI. Avoid clearing live leases or editing job state by hand.

During a retailer data outage, continue serving stored observations with their observation dates. Temporary errors and throttling back off for up to five attempts. Repeated or permanent failures remain visible on the job. Manual prices and CSV imports remain available. During a Groq outage, the financial assistant uses its deterministic fallback. Check provider health and quota before rotating credentials or retrying jobs.

## Pre-release privacy and device checks

- Android notification access is opt-in and constrained to selected packages. Confirm on a real device that raw notifications are discarded, the structured queue survives restart, revocation works, and retries do not post duplicate expenses. Verify the HTTPS production hostname is baked into the signed build and keep signing credentials outside Git.
- The financial assistant sends qualitative statements to Groq. The separately invoked expense text and document parsers can send submitted text or receipt images to Groq. Review provider processing, user disclosures, and retention for both flows before enabling Groq for a public release. Manual entry and the financial assistant fallback work without the key.
- Confirm no credentials, raw notifications, financial payloads, or session tokens appear in app, ingress, or provider logs. Review account deletion, backup retention, and full data export before public signup. The existing Profile download is a workspace snapshot.

## Staging load rehearsal

Run k6 from a machine that can reach the private staging Go service and the public staging site. Use the local ignored `deploy/load.js` with staging credentials supplied through the runner environment. Keep the k6 file and fixture values out of Git. Use disposable accounts and data so a repeat run cannot change a real ledger.

| Scenario | Variables in addition to `BUYWISE_LOAD_ENV=staging`, `WEB_URL`, and `PRIVATE_API_URL` |
| --- | --- |
| `auth` | `LOAD_EMAIL` and `LOAD_PASSWORD` for a synthetic account. This scenario opens and revokes sessions at a controlled rate. |
| `core` | `BUYWISE_TOKENS`, `GOAL_ID`, `PRODUCT_ID`, `OFFER_ID`, `ACCOUNT_ID`, `EXPENSE_CATEGORY_ID`, a new `RUN_ID`, and `EXPENSE_DATE` in `YYYY-MM-DD`. Use sessions and objects from the same fixture account. |
| `ai` | `BUYWISE_TOKENS`. Start with one virtual user, or provide one session from a distinct fixture account per concurrent virtual user. |

Set `SCENARIO`, then run `k6 run deploy/load.js` for each scenario. Set `VUS` for core and AI, `AUTH_RATE` for auth arrivals per 10 seconds, and `DURATION` for any run. Collect the release SHA, environment, fixture size, workload, duration, p95 per scenario, request failures, checks, resource use, first bottleneck, and any observed provider fallback. Initial gates are under 1% failed requests, over 99% successful checks, and p95 under 1 s for auth and core or under 7 s for AI. Check the synthetic ledger for one entry per idempotency key.

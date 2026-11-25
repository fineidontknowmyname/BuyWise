# BuyWise release rehearsal

Use a candidate commit that has passed GitHub Release gates. Deploy it to the isolated staging environment described in the [deployment runbook](README.md). Verify that the web hostname, Android build, private API, database, provider keys and backups all belong to staging before creating fixtures. Use the same candidate commit for the web and Android checks.

## Seed staging through the app

1. On the staging sign-in page, create two new accounts. Use fresh addresses such as `rehearsal-<run-id>@example.test` and `observer-<run-id>@example.test`, with unique passwords stored in the staging secret manager. Use the observer for isolation checks. Label fixture names and notes **Rehearsal**. Record the run ID in the evidence without the credentials.
2. Sign in to the first account. In **Accounts**, create **Rehearsal Payment** with ₹1,000, **Rehearsal Savings** with ₹1,000, and **Rehearsal Reserve** with ₹500. In the financial profile, designate Savings as general savings and Reserve as the emergency reserve.
3. In the financial profile, set a ₹60,000 monthly fixed income, ₹20,000 monthly rent, ₹500 emergency target, ₹5,000 minimum buffer, 10% buffer, 70% variable-income allowance and debt status **Current**. Start with an empty ledger. Use new accounts for each rehearsal because purchase completion creates permanent ledger history.
4. Create a ₹1,000 **Rehearsal Headphones** goal and link a product with variant **Black**. Create a second ₹1,000 goal named **Rehearsal Monitor**. Use the staging market date in India for prices and expenses. Use a manual offer so the rehearsal does not depend on retailer credentials or real transactions.

## Verify the web and Android journeys

| Journey | Action and expected result |
| --- | --- |
| Onboarding | Sign out and sign back in. Confirm the dashboard loads and includes the new accounts. Open the headphone goal link using the observer session. The observer must be unable to read or change the goal. |
| Planning | Save a simple monthly headphone plan of ₹800. Add a ₹900 contribution backed by Rehearsal Savings. The goal should show ₹900 funded and the savings account's ledger balance should remain ₹1,000. |
| Tracking and decision | Add an offer for the same variant and record an in-stock landed price of ₹600 for today. Confirm the observation appears with the correct amount, date, and manual source. Save an assessment and confirm it appears in decision history. |
| Purchase | Use **Mark purchased** and enter ₹600, today's date, Rehearsal Payment, and the Shopping category. Preview should show ₹300 surplus. Complete once and reload. The goal should be completed and the payment ledger should have one ₹600 purchase expense. A replay with the same purchase key should return the original transaction. |
| Surplus allocation | Allocate ₹100 to Rehearsal Monitor from Savings and ₹150 to Rehearsal Reserve. Expect ₹50 remaining, ₹100 funded on the monitor goal, ₹650 in Reserve, and ₹250 in Savings. Payment should return to ₹1,000 after the funding transfer and purchase expense. Verify the Activity page reflects the changes. |
| Mobile expense | Sign in to the staging Android build. Enter a ₹25 manual expense from Payment. Verify the same entry on web. With a staged identical retry, verify a single ₹25 debit. |
| Mobile capture | On a device with a synthetic notification source, grant access and select only that source. Deliver a ₹45 INR debit. Restart BuyWise before syncing and confirm the suggestion survives. Sync and verify Payment remains ₹975 until confirmation. Confirm the suggestion and expect ₹930. Retry the delivery or confirmation and verify Payment remains ₹930. Disable capture and revoke access, then verify new notifications produce no suggestions. |
| Session and export | Sign out on each client and verify that client's session can no longer read data. From a fresh browser session download the Profile workspace snapshot and confirm it contains only the signed-in workspace. |

Build the APK with Node 22, JDK 21 and Android SDK 36. From the repo root in PowerShell, use the staging HTTPS origin in `BUYWISE_SERVER_URL`. Use a debug APK for the internal rehearsal and verify the signed release separately on a physical device.

```powershell
$env:BUYWISE_SERVER_URL = 'https://YOUR_STAGING_HOST'
npm.cmd --prefix frontend/mobile ci
pnpm.cmd --dir frontend build:mobile
npm.cmd --prefix frontend/mobile run sync
Push-Location frontend/mobile/android
.\gradlew.bat assembleDebug
Pop-Location
```

## Rehearse recovery and prepare support

1. Confirm public `/health` and private `/readyz` return `200`. Confirm the private metrics reader records requests and no Go metrics are reachable on the public hostname. Test the alert destination with the operator on call and record receipt.
2. On staging, restart the price worker during a queued refresh and verify recovery using the [worker procedure](README.md#worker-and-service-monitoring). Use the normal operator path to inspect status and avoid repeatedly queuing the same job during an outage.
3. Make a backup after the fixture journey and restore it into an isolated database. Start a recovery instance and verify health, the ₹930 payment balance, ₹50 remaining surplus, and exactly one purchase expense. Record the age of the backup and elapsed recovery time. Rehearse an unhealthy API response and confirm the web health check indicates the dependency failure.
4. Keep the last compatible web/API release and its commit or image IDs available. On a candidate regression, stop promotion, preserve the current database and compare applied migrations with the previous release. Roll back both application services only when their older code supports the current schema and its recorded purchases. If the schema or data changed incompatibly, fix forward. If database recovery is required, restore into an isolated instance, verify the ledger and agree on the recovery point before switching connections.
5. Assign a release owner and a support contact. For a report, collect the time with timezone, platform and build, affected screen, expected behavior, HTTP status and request ID when available. Keep credentials, notification bodies, receipt images and account data out of tickets. For a duplicate charge report, verify existing transaction IDs and balances before asking the user to retry, void, or enter another transaction.

## Candidate release notes

BuyWise brings account balances, spending budgets, purchase plans and price history together. Users can save affordability assessments, record completed purchases and allocate unused funds. Android supports manual expenses and opt-in notification suggestions, with review before a suggestion becomes an expense.

- The Profile export is a workspace snapshot. Complete account export, account deletion and the retention policy require closure before public signup.
- Price refresh depends on provider configuration, quota and supported listings. Manual prices and CSV imports cover unsupported listings and provider outages. With no Groq key, the financial assistant uses its built-in explanation.
- Posting to the ledger requires a connection. Android notification parsing supports selected apps and recognizable INR debit messages. Use manual entry for unsupported formats.

## Local rehearsal record

On 20 November 2026, the application at `54023b0` was rehearsed against disposable PostgreSQL databases with synthetic accounts. The packaged web server handled registration, sign-in, the secured session cookie, and goal creation. Planning, tenant isolation, manual price history, a saved decision, purchase and allocation retries, and mobile gateway expense review all passed. The test reconciled the ₹930 payment balance and ₹50 remaining surplus after backup and restore. The health endpoint returned `503` while the API was stopped and recovered after the API started on the restore database.

The production web build, Android web bundle and Capacitor sync passed. Gradle could not download its distribution on the rehearsal machine. APK build and device sign-off remain pending, along with staging TLS, live worker recovery, alert delivery and the Phase 11 load and CI evidence.

## Sign off the candidate

Record the commit SHA, staging hostname, run ID, device and OS, APK identifier, operator, date, journey outcomes, CI link, alert receipt, restore evidence and k6 results. Mark each pending item and assign an owner. Have a second operator complete the runbook unaided. If the demo requires a live walkthrough, record a backup walkthrough using these staging fixtures and store it with the demonstration materials. Keep the candidate pending until the release owner approves the staging, recovery, load, device and privacy evidence.

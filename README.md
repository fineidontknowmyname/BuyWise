# BuyWise

> **Plan smarter. Buy wiser.**

BuyWise is a modern personal finance and purchase planning web application designed to help you organize accounts, manage discretionary spending budgets, and evaluate purchase goals with intelligent decision rules.

---

## Architecture

- **Backend:** Go 1.27, standard library `net/http` web service bound to `127.0.0.1:8080`, `pgx` driver, plain SQL migrations.
- **Frontend:** SvelteKit 3, Svelte 5 (Runes), TypeScript, Vite, custom fintech design system with light/dark theme support.
- **Database:** Local PostgreSQL 18.
- **Security & Privacy:** The browser communicates exclusively with SvelteKit. SvelteKit server endpoints proxy requests to the private Go API on loopback (`127.0.0.1`).

---

## Quickstart

### 1. Database Setup
Create the PostgreSQL databases:
```powershell
createdb -U postgres selfemi
createdb -U postgres selfemi_test
```

### 2. Environment Configuration
Copy `backend/.env.example` to `backend/.env` and update your PostgreSQL credentials:
```env
DATABASE_URL=postgres://postgres:<PASSWORD>@127.0.0.1:5432/selfemi?sslmode=disable
API_ADDR=127.0.0.1:8080
API_URL=http://127.0.0.1:8080
```

### 3. Start the Application
From the repository root:
```powershell
pnpm dev
```
Open **[http://localhost:5173](http://localhost:5173)**. Both the Go backend API and the SvelteKit frontend run concurrently.

---

## Core Features

- **Fintech Overview Dashboard:** Real-time summary of liquid balances, active purchase goals, monthly budget utilization, interactive SVG financial charts, and recent transaction history.
- **Purchase Goals:** Plan and track expensive milestone purchases with rule-based affordability decisions (`BUY`, `WAIT`, `SAVE MORE`, `REASSESS`).
- **Account Ledger:** Track cash, bank, and savings accounts with positive/negative balances and voidable transaction entries.
- **Committed vs. Discretionary Categories:** Distinguish fixed obligations (rent, debt EMI) from flexible spending to calculate true discretionary savings capacity.
- **Monthly Budgets:** Establish category spending limits and monitor utilization progress bars with adaptive threshold alerts.
- **Theme System:** Full support for Light and Dark themes, with system preference detection and persistent preference storage.

---

## Developer Commands

| Command | Action |
|---|---|
| `pnpm dev` | Run database migrations, then start Go API and SvelteKit frontend |
| `pnpm test` | Run Go test suite and Vitest frontend unit tests |
| `pnpm lint` | Run Go formatting, `go vet`, Prettier, ESLint, and `svelte-check` |
| `pnpm format` | Auto-format Go code and frontend components |
| `pnpm migrate` | Run database migrations only |

## Purchase completion

Open a goal and choose **Mark purchased**. Enter the date, seller, landed cost, payment account and expense category. The review shows the expense, any earmarked funds to transfer to the payment account, and the surplus that will remain reserved. A shortfall uses unallocated money in the selected account only when the saved financial profile and current balances support it.

After completion, allocate surplus to another active goal, general savings, an emergency reserve, or a split across those destinations. A transfer between accounts changes both balances. Allocating in the same account changes only the earmark. The original target and contribution history stay available alongside the expense and allocation history.

All routes below require a tenant session. Amounts use integer paise and dates use `YYYY-MM-DD` in India.

| Route | Contract |
|---|---|
| `GET /goals/{id}/purchase` | Returns the purchase record if present, unallocated surplus, available destination options, and allocation history. |
| `POST /goals/{id}/purchase/preview` | Accepts `date`, `seller`, `costPaise`, `reference`, `paymentAccountId`, and `categoryId`. Returns `summary` and a review `token`. |
| `POST /goals/{id}/purchase` | Accepts the same fields plus the `token`, a UUID `key`, and `confirmed: true`. Returns `201` on creation, `200` on an identical retry, or `409` if the review changed or a different purchase is already recorded. |
| `POST /goals/{id}/purchase/allocations` | Accepts a UUID `key`, `confirmed: true`, and up to ten `lines` with `accountId`, `amountPaise`, and optional `goalId`. Returns `201` on creation and `200` on an identical retry. |

Migration `0015` adds the append-only purchase and allocation records and marks their ledger entries as purchase-managed. Apply it before starting the updated API. Rolling back only the application after purchases have been recorded would remove access to the lifecycle and its surplus protections. Recover by restoring a consistent pre-upgrade database and application backup, or retain the migration and fix forward.

## Expense capture

Open **Expenses** for manual entry and the suggestion inbox. The inbox shows the notification source, a category suggestion when the merchant matches a rule, and possible duplicates. You can change any ledger field before confirmation or dismiss the suggestion. Confirmation checks the selected account again for a matching expense. Recording a separate expense for the same date and amount requires explicit confirmation.

| Authenticated route | Contract |
|---|---|
| `POST /expenses` | Accepts `accountId`, `categoryId`, integer `amountPaise`, `occurredOn` in `YYYY-MM-DD`, and optional `note`. Requires `Idempotency-Key`. A matching retry returns the recorded expense. |
| `GET /expenses/candidates` | Returns up to 100 pending suggestions, oldest first. Reviewing them makes room for later suggestions. |
| `POST /expenses/candidates/notifications` | Accepts structured INR debits with `sourceApp`, integer `amountPaise`, `currency: "INR"`, RFC 3339 `occurredAt`, and optional `merchant` and `reference`. Returns `201` for a new suggestion or `200` for a matching delivery. |
| `POST /expenses/candidates/{id}/confirm` | Accepts the editable expense fields, `confirmed: true`, and optional `allowDuplicate: true` for a separate expense. A matching retry returns the same ledger entry. |
| `POST /expenses/candidates/{id}/reject` | Dismisses a pending suggestion and returns `204`. Retrying a dismissal returns `204`. |

Migration `0016` adds the inbox. Ingestion stores the source app, date, amount, bounded merchant name and deduplication hashes. It accepts no raw notification body. Review clears the suggested merchant from the inbox record and keeps the status, timestamps, deduplication hashes and optional ledger link. The native listener, on-device parsing and permission opt-in belong to the Android integration in Phase 10. The web app requests no notification permission.

## Financial assistant

Open **Financial Assistant** and select a question. The page reads existing purchase goals, plan projections, saved decisions, retailer offer history, monthly budgets, and financial profile estimates. Each answer links back to its BuyWise records. Reports, the simulator, and the product page use the same assistant. Create and edit purchase goals through **Purchase Goals**.

`POST /api/ai/chat` is served by the SvelteKit app, normally on port `5173`. It accepts a signed-in browser session or a BuyWise bearer session for native clients. Browser requests with a session cookie require the app's origin. The request body is JSON.

| `topic` | Additional fields |
|---|---|
| `capacity`, `budget` | None |
| `goal`, `decision` | `goalId` |
| `price` | `productId` |

Responses contain `title`, `facts`, `sources`, `explanation`, and `mode`. `ai_assisted` means Groq prioritized statements written by BuyWise. BuyWise renders those statements and the original records. `buywise` means the explanation uses BuyWise's default order. The optional `fallback` field distinguishes missing AI configuration from a temporary provider failure. A missing or inaccessible record returns `404`, and a source API failure returns `503`.

Assistant history lives in the page for the current visit. BuyWise keeps no stored transcript. The provider receives only the selected topic and qualitative statements. Account details, record titles, IDs, transaction notes, and financial figures stay within BuyWise. The application logs only the topic, outcome, fact count, and elapsed time for answered requests. The endpoint limits requests per signed-in user, and the Groq call has a bounded timeout and retry. The assistant works without a Groq key and requires no database migration.

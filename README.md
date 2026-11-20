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

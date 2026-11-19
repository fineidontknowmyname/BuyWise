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

CREATE SCHEMA IF NOT EXISTS finance;

CREATE TABLE finance.accounts (
	id uuid PRIMARY KEY,
	name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
	opening_balance_paise bigint NOT NULL
		CHECK (opening_balance_paise BETWEEN -1000000000000 AND 1000000000000),
	created_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX accounts_name_unique ON finance.accounts (lower(name));

CREATE TABLE finance.categories (
	id uuid PRIMARY KEY,
	name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
	kind text NOT NULL CHECK (kind IN ('income', 'expense')),
	committed boolean NOT NULL DEFAULT false,
	created_at timestamptz NOT NULL DEFAULT now(),
	CHECK (kind = 'expense' OR committed = false)
);

CREATE UNIQUE INDEX categories_kind_name_unique ON finance.categories (kind, lower(name));

CREATE TABLE finance.transactions (
	id uuid PRIMARY KEY,
	account_id uuid NOT NULL REFERENCES finance.accounts (id),
	category_id uuid NOT NULL REFERENCES finance.categories (id),
	kind text NOT NULL CHECK (kind IN ('income', 'expense')),
	amount_paise bigint NOT NULL CHECK (amount_paise BETWEEN 1 AND 1000000000000),
	occurred_on date NOT NULL,
	note text NOT NULL DEFAULT '' CHECK (char_length(note) <= 500),
	idempotency_key text NOT NULL UNIQUE CHECK (char_length(idempotency_key) BETWEEN 1 AND 100),
	voided_at timestamptz,
	void_reason text NOT NULL DEFAULT '' CHECK (char_length(void_reason) <= 500),
	created_at timestamptz NOT NULL
);

CREATE INDEX transactions_account_date ON finance.transactions (account_id, occurred_on DESC, id DESC);

INSERT INTO finance.categories (id, name, kind, committed) VALUES
	(uuidv7(), 'Salary', 'income', false),
	(uuidv7(), 'Other income', 'income', false),
	(uuidv7(), 'Rent', 'expense', true),
	(uuidv7(), 'Utilities', 'expense', true),
	(uuidv7(), 'Insurance', 'expense', true),
	(uuidv7(), 'Loan payments', 'expense', true),
	(uuidv7(), 'Groceries', 'expense', false),
	(uuidv7(), 'Transport', 'expense', false),
	(uuidv7(), 'Dining out', 'expense', false),
	(uuidv7(), 'Shopping', 'expense', false),
	(uuidv7(), 'Entertainment', 'expense', false),
	(uuidv7(), 'Other expense', 'expense', false);

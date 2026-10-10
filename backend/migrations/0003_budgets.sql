CREATE TABLE finance.budgets (
	category_id uuid PRIMARY KEY REFERENCES finance.categories (id),
	limit_paise bigint NOT NULL CHECK (limit_paise BETWEEN 0 AND 1000000000000),
	updated_at timestamptz NOT NULL
);

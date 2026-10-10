CREATE SCHEMA IF NOT EXISTS goals;

CREATE TABLE goals.goals (
	id uuid PRIMARY KEY,
	name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
	created_at timestamptz NOT NULL
);

ALTER TABLE finance.categories ADD COLUMN IF NOT EXISTS archived boolean NOT NULL DEFAULT false;

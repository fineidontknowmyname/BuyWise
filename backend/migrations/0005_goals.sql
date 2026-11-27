ALTER TABLE goals.goals ADD COLUMN IF NOT EXISTS target_price_paise bigint NOT NULL DEFAULT 0 CHECK (target_price_paise >= 0);
ALTER TABLE goals.goals ADD COLUMN IF NOT EXISTS saved_amount_paise bigint NOT NULL DEFAULT 0 CHECK (saved_amount_paise >= 0);
ALTER TABLE goals.goals ADD COLUMN IF NOT EXISTS target_date date;
ALTER TABLE goals.goals ADD COLUMN IF NOT EXISTS priority text NOT NULL DEFAULT 'medium' CHECK (priority IN ('high', 'medium', 'low'));
ALTER TABLE goals.goals ADD COLUMN IF NOT EXISTS product_id uuid;
ALTER TABLE goals.goals ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'cancelled'));

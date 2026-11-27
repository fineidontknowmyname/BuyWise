-- 0007_product_engine.sql: Extended Product Profile, Price Observation Details, and Document Upload Tracking

-- 1. Extend pricing.products
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS variant text NOT NULL DEFAULT '' CHECK (char_length(variant) <= 100);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS original_price_paise bigint NOT NULL DEFAULT 0 CHECK (original_price_paise >= 0);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS discount_percent numeric(5,2) NOT NULL DEFAULT 0 CHECK (discount_percent >= 0 AND discount_percent <= 100);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS currency text NOT NULL DEFAULT 'INR' CHECK (char_length(currency) <= 10);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS availability text NOT NULL DEFAULT 'in_stock' CHECK (char_length(availability) <= 50);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS specifications jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS last_retrieved_at timestamptz;
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'manual' CHECK (char_length(source) <= 50);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS verification_status text NOT NULL DEFAULT 'unverified' CHECK (char_length(verification_status) <= 50);
ALTER TABLE pricing.products ADD COLUMN IF NOT EXISTS warnings jsonb NOT NULL DEFAULT '[]'::jsonb;

-- 2. Extend pricing.price_observations
ALTER TABLE pricing.price_observations ADD COLUMN IF NOT EXISTS goal_id uuid REFERENCES goals.goals (id) ON DELETE SET NULL;
ALTER TABLE pricing.price_observations ADD COLUMN IF NOT EXISTS currency text NOT NULL DEFAULT 'INR' CHECK (char_length(currency) <= 10);
ALTER TABLE pricing.price_observations ADD COLUMN IF NOT EXISTS price_type text NOT NULL DEFAULT 'regular' CHECK (price_type IN ('regular', 'discounted', 'original'));
ALTER TABLE pricing.price_observations ADD COLUMN IF NOT EXISTS retailer text NOT NULL DEFAULT '' CHECK (char_length(retailer) <= 100);
ALTER TABLE pricing.price_observations ADD COLUMN IF NOT EXISTS source_ref text NOT NULL DEFAULT '' CHECK (char_length(source_ref) <= 2000);
ALTER TABLE pricing.price_observations ADD COLUMN IF NOT EXISTS notes text NOT NULL DEFAULT '' CHECK (char_length(notes) <= 1000);

-- 3. Create pricing.document_uploads
CREATE TABLE IF NOT EXISTS pricing.document_uploads (
	id uuid PRIMARY KEY,
	goal_id uuid NOT NULL REFERENCES goals.goals (id) ON DELETE CASCADE,
	product_id uuid REFERENCES pricing.products (id) ON DELETE SET NULL,
	filename text NOT NULL CHECK (char_length(filename) BETWEEN 1 AND 255),
	file_type text NOT NULL CHECK (char_length(file_type) <= 20),
	file_size bigint NOT NULL CHECK (file_size >= 0),
	purpose text NOT NULL DEFAULT 'prices' CHECK (char_length(purpose) <= 50),
	status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'previewed', 'confirmed', 'failed')),
	parsed_data jsonb NOT NULL DEFAULT '{}'::jsonb,
	summary jsonb NOT NULL DEFAULT '{}'::jsonb,
	created_at timestamptz NOT NULL,
	confirmed_at timestamptz
);

CREATE INDEX IF NOT EXISTS document_uploads_goal ON pricing.document_uploads (goal_id, created_at DESC);

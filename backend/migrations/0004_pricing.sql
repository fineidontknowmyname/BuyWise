CREATE SCHEMA IF NOT EXISTS pricing;

CREATE TABLE pricing.products (
	id uuid PRIMARY KEY,
	name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
	brand text NOT NULL DEFAULT '' CHECK (char_length(brand) <= 100),
	model text NOT NULL DEFAULT '' CHECK (char_length(model) <= 100),
	category text NOT NULL DEFAULT '' CHECK (char_length(category) <= 100),
	current_price_paise bigint NOT NULL DEFAULT 0 CHECK (current_price_paise >= 0),
	target_price_paise bigint NOT NULL DEFAULT 0 CHECK (target_price_paise >= 0),
	url text NOT NULL DEFAULT '' CHECK (char_length(url) <= 2000),
	image_url text NOT NULL DEFAULT '' CHECK (char_length(image_url) <= 2000),
	retailer text NOT NULL DEFAULT '' CHECK (char_length(retailer) <= 100),
	notes text NOT NULL DEFAULT '' CHECK (char_length(notes) <= 1000),
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL
);

CREATE TABLE pricing.price_import_batches (
	id uuid PRIMARY KEY,
	product_id uuid NOT NULL REFERENCES pricing.products (id) ON DELETE CASCADE,
	filename text NOT NULL CHECK (char_length(filename) BETWEEN 1 AND 255),
	total_rows int NOT NULL CHECK (total_rows >= 0),
	imported_rows int NOT NULL CHECK (imported_rows >= 0),
	duplicate_rows int NOT NULL CHECK (duplicate_rows >= 0),
	error_rows int NOT NULL CHECK (error_rows >= 0),
	created_at timestamptz NOT NULL
);

CREATE TABLE pricing.price_observations (
	id uuid PRIMARY KEY,
	product_id uuid NOT NULL REFERENCES pricing.products (id) ON DELETE CASCADE,
	price_paise bigint NOT NULL CHECK (price_paise > 0 AND price_paise <= 1000000000000),
	observed_on date NOT NULL,
	source text NOT NULL DEFAULT 'manual' CHECK (char_length(source) <= 50),
	provenance text NOT NULL DEFAULT 'manual' CHECK (char_length(provenance) <= 50),
	import_batch_id uuid REFERENCES pricing.price_import_batches (id) ON DELETE SET NULL,
	created_at timestamptz NOT NULL
);

CREATE INDEX price_observations_product_date ON pricing.price_observations (product_id, observed_on DESC, id DESC);
CREATE UNIQUE INDEX price_observations_product_date_unique ON pricing.price_observations (product_id, observed_on);

CREATE TABLE pricing.retailer_offers (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    product_id uuid NOT NULL,
    retailer text NOT NULL CHECK (char_length(retailer) BETWEEN 1 AND 100),
    seller text NOT NULL CHECK (char_length(seller) <= 150),
    listing_url text NOT NULL CHECK (char_length(listing_url) <= 2000),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, id),
    FOREIGN KEY (user_id, product_id) REFERENCES pricing.products (user_id, id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX retailer_offers_url ON pricing.retailer_offers (user_id, listing_url) WHERE listing_url <> '';
CREATE UNIQUE INDEX retailer_offers_manual ON pricing.retailer_offers
    (user_id, product_id, lower(retailer), lower(seller)) WHERE listing_url = '';
CREATE INDEX retailer_offers_product ON pricing.retailer_offers (user_id, product_id);

CREATE TABLE pricing.offer_observations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    offer_id uuid NOT NULL,
    observed_on date NOT NULL,
    observed_at timestamptz,
    fetched_at timestamptz,
    recorded_at timestamptz NOT NULL DEFAULT now(),
    source text NOT NULL CHECK (source IN ('manual', 'csv', 'provider')),
    source_label text NOT NULL CHECK (char_length(source_label) <= 150),
    availability text NOT NULL CHECK (availability IN ('unknown', 'in_stock', 'out_of_stock', 'pre_order')),
    item_paise bigint NOT NULL CHECK (item_paise BETWEEN 1 AND 1000000000000),
    tax_paise bigint CHECK (tax_paise BETWEEN 0 AND 1000000000000),
    delivery_paise bigint CHECK (delivery_paise BETWEEN 0 AND 1000000000000),
    discount_paise bigint CHECK (discount_paise BETWEEN 0 AND 1000000000000),
    CHECK (COALESCE(discount_paise, 0) < item_paise + COALESCE(tax_paise, 0) + COALESCE(delivery_paise, 0)),
    UNIQUE NULLS NOT DISTINCT (user_id, offer_id, observed_on, observed_at, source, item_paise, tax_paise, delivery_paise, discount_paise, availability),
    FOREIGN KEY (user_id, offer_id) REFERENCES pricing.retailer_offers (user_id, id) ON DELETE CASCADE
);

CREATE INDEX offer_observations_history ON pricing.offer_observations (user_id, offer_id, observed_on DESC, observed_at DESC, recorded_at DESC);

CREATE TABLE pricing.target_thresholds (
    user_id uuid NOT NULL DEFAULT identity.current_user_id(),
    product_id uuid NOT NULL,
    ideal_paise bigint CHECK (ideal_paise BETWEEN 1 AND 1000000000000),
    good_paise bigint CHECK (good_paise BETWEEN 1 AND 1000000000000),
    maximum_paise bigint CHECK (maximum_paise BETWEEN 1 AND 1000000000000),
    PRIMARY KEY (user_id, product_id),
    FOREIGN KEY (user_id, product_id) REFERENCES pricing.products (user_id, id) ON DELETE CASCADE,
    CHECK (ideal_paise <= good_paise),
    CHECK (good_paise <= maximum_paise),
    CHECK (ideal_paise <= maximum_paise)
);

ALTER TABLE pricing.retailer_offers ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.retailer_offers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.retailer_offers
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE pricing.offer_observations ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.offer_observations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.offer_observations
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

ALTER TABLE pricing.target_thresholds ENABLE ROW LEVEL SECURITY;
ALTER TABLE pricing.target_thresholds FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON pricing.target_thresholds
    USING (user_id = identity.current_user_id())
    WITH CHECK (user_id = identity.current_user_id());

GRANT SELECT, INSERT ON pricing.retailer_offers, pricing.offer_observations TO buywise_tenant;
GRANT SELECT, INSERT, UPDATE ON pricing.target_thresholds TO buywise_tenant;

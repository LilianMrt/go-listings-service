CREATE TYPE listing_status AS ENUM ('draft', 'published', 'sold');

CREATE TABLE listings (
    id          UUID PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_cents BIGINT NOT NULL,
    currency    TEXT NOT NULL DEFAULT 'EUR',
    city        TEXT NOT NULL,
    postal_code TEXT NOT NULL,
    surface_m2  INTEGER NOT NULL DEFAULT 0,
    rooms       INTEGER NOT NULL DEFAULT 0,
    status      listing_status NOT NULL DEFAULT 'draft',
    seller_id   UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_listings_city ON listings (city);
CREATE INDEX idx_listings_status ON listings (status);
CREATE INDEX idx_listings_price ON listings (price_cents);

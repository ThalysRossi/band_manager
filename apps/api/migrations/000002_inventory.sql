-- +goose Up
CREATE TABLE merch_products (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('shirt', 'hoodie', 'tote_bag', 'patch', 'sticker', 'vinyl', 'cd', 'cassette', 'accessory')),
    photo_object_key TEXT NOT NULL,
    photo_content_type TEXT NOT NULL,
    photo_size_bytes INTEGER NOT NULL CHECK (photo_size_bytes > 0),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE merch_variants (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    product_id UUID NOT NULL REFERENCES merch_products (id),
    size TEXT NOT NULL CHECK (size IN ('not_applicable', 'one_size', 'pp', 'p', 'm', 'g', 'gg', 'xgg')),
    colour TEXT NOT NULL,
    normalized_colour TEXT NOT NULL,
    price_amount INTEGER NOT NULL CHECK (price_amount >= 0),
    cost_amount INTEGER NOT NULL CHECK (cost_amount >= 0),
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    quantity INTEGER NOT NULL CHECK (quantity >= 0),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    product_id UUID NOT NULL REFERENCES merch_products (id),
    variant_id UUID NOT NULL REFERENCES merch_variants (id),
    movement_type TEXT NOT NULL CHECK (movement_type IN ('initial_stock', 'manual_adjustment')),
    quantity_delta INTEGER NOT NULL,
    quantity_after INTEGER NOT NULL CHECK (quantity_after >= 0),
    reason TEXT NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX merch_products_active_identity_idx
    ON merch_products (band_id, category, normalized_name)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX merch_variants_active_identity_idx
    ON merch_variants (product_id, size, normalized_colour)
    WHERE deleted_at IS NULL;

CREATE INDEX merch_products_band_active_idx
    ON merch_products (band_id, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX merch_variants_product_active_idx
    ON merch_variants (product_id, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX inventory_movements_variant_created_at_idx
    ON inventory_movements (variant_id, created_at);

-- +goose Down
DROP TABLE inventory_movements;
DROP TABLE merch_variants;
DROP TABLE merch_products;

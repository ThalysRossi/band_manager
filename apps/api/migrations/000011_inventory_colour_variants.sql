-- +goose Up
CREATE TEMP TABLE inventory_colour_variant_migration_guard (
    ready BOOLEAN NOT NULL CONSTRAINT reset_local_merch_before_colour_variant_migration CHECK (ready)
);

INSERT INTO inventory_colour_variant_migration_guard (ready)
SELECT NOT EXISTS (SELECT 1 FROM merch_products)
    AND NOT EXISTS (SELECT 1 FROM sales)
    AND NOT EXISTS (SELECT 1 FROM payments);

DROP TABLE inventory_colour_variant_migration_guard;

CREATE TABLE merch_colour_variants (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    product_id UUID NOT NULL REFERENCES merch_products (id),
    colour TEXT NOT NULL CHECK (length(btrim(colour)) > 0),
    normalized_colour TEXT NOT NULL CHECK (length(btrim(normalized_colour)) > 0),
    price_amount INTEGER NOT NULL CHECK (price_amount >= 0),
    cost_amount INTEGER NOT NULL CHECK (cost_amount >= 0),
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    photo_full_object_key TEXT NOT NULL CHECK (length(btrim(photo_full_object_key)) > 0),
    photo_full_content_type TEXT NOT NULL CHECK (photo_full_content_type = 'image/webp'),
    photo_full_size_bytes INTEGER NOT NULL CHECK (photo_full_size_bytes > 0 AND photo_full_size_bytes <= 10485760),
    photo_full_width INTEGER NOT NULL CHECK (photo_full_width > 0 AND photo_full_width <= 3840),
    photo_full_height INTEGER NOT NULL CHECK (photo_full_height > 0 AND photo_full_height <= 3840),
    photo_display_object_key TEXT NOT NULL CHECK (length(btrim(photo_display_object_key)) > 0),
    photo_display_content_type TEXT NOT NULL CHECK (photo_display_content_type = 'image/webp'),
    photo_display_size_bytes INTEGER NOT NULL CHECK (photo_display_size_bytes > 0 AND photo_display_size_bytes <= 2097152),
    photo_display_width INTEGER NOT NULL CHECK (photo_display_width > 0 AND photo_display_width <= 1280),
    photo_display_height INTEGER NOT NULL CHECK (photo_display_height > 0 AND photo_display_height <= 960),
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX merch_colour_variants_active_identity_idx
    ON merch_colour_variants (product_id, normalized_colour)
    WHERE deleted_at IS NULL;

ALTER TABLE merch_variants
    ADD COLUMN colour_variant_id UUID NOT NULL REFERENCES merch_colour_variants (id);

DROP INDEX merch_variants_active_identity_idx;

CREATE UNIQUE INDEX merch_variants_active_identity_idx
    ON merch_variants (colour_variant_id, size)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX merch_variants_active_identity_idx;

CREATE UNIQUE INDEX merch_variants_active_identity_idx
    ON merch_variants (product_id, size, normalized_colour)
    WHERE deleted_at IS NULL;

ALTER TABLE merch_variants DROP COLUMN colour_variant_id;
DROP TABLE merch_colour_variants;

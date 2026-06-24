-- +goose Up
ALTER TABLE merch_products
    DROP CONSTRAINT merch_products_photo_object_key_present_check;

ALTER TABLE merch_products
    DROP CONSTRAINT merch_products_photo_content_type_present_check;

ALTER TABLE merch_products
    RENAME COLUMN photo_object_key TO photo_full_object_key;

ALTER TABLE merch_products
    RENAME COLUMN photo_content_type TO photo_full_content_type;

ALTER TABLE merch_products
    RENAME COLUMN photo_size_bytes TO photo_full_size_bytes;

ALTER TABLE merch_products
    ADD COLUMN photo_full_width INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN photo_full_height INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN photo_display_object_key TEXT NOT NULL DEFAULT 'legacy-display.webp',
    ADD COLUMN photo_display_content_type TEXT NOT NULL DEFAULT 'image/webp',
    ADD COLUMN photo_display_size_bytes INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN photo_display_width INTEGER NOT NULL DEFAULT 4,
    ADD COLUMN photo_display_height INTEGER NOT NULL DEFAULT 3;

UPDATE merch_products
SET photo_full_content_type = 'image/webp'
WHERE photo_full_content_type <> 'image/webp';

ALTER TABLE merch_products
    ADD CONSTRAINT merch_products_photo_full_object_key_present_check
    CHECK (length(btrim(photo_full_object_key)) > 0),
    ADD CONSTRAINT merch_products_photo_full_content_type_check
    CHECK (photo_full_content_type = 'image/webp'),
    ADD CONSTRAINT merch_products_photo_full_size_bytes_check
    CHECK (photo_full_size_bytes > 0 AND photo_full_size_bytes <= 10485760),
    ADD CONSTRAINT merch_products_photo_full_width_check
    CHECK (photo_full_width > 0 AND photo_full_width <= 3840),
    ADD CONSTRAINT merch_products_photo_full_height_check
    CHECK (photo_full_height > 0 AND photo_full_height <= 3840),
    ADD CONSTRAINT merch_products_photo_display_object_key_present_check
    CHECK (length(btrim(photo_display_object_key)) > 0),
    ADD CONSTRAINT merch_products_photo_display_content_type_check
    CHECK (photo_display_content_type = 'image/webp'),
    ADD CONSTRAINT merch_products_photo_display_size_bytes_check
    CHECK (photo_display_size_bytes > 0 AND photo_display_size_bytes <= 2097152),
    ADD CONSTRAINT merch_products_photo_display_width_check
    CHECK (photo_display_width > 0 AND photo_display_width <= 1280),
    ADD CONSTRAINT merch_products_photo_display_height_check
    CHECK (photo_display_height > 0 AND photo_display_height <= 960),
    ADD CONSTRAINT merch_products_photo_display_aspect_ratio_check
    CHECK (photo_display_width * 3 = photo_display_height * 4);

-- +goose Down
ALTER TABLE merch_products
    DROP CONSTRAINT merch_products_photo_display_aspect_ratio_check,
    DROP CONSTRAINT merch_products_photo_display_height_check,
    DROP CONSTRAINT merch_products_photo_display_width_check,
    DROP CONSTRAINT merch_products_photo_display_size_bytes_check,
    DROP CONSTRAINT merch_products_photo_display_content_type_check,
    DROP CONSTRAINT merch_products_photo_display_object_key_present_check,
    DROP CONSTRAINT merch_products_photo_full_height_check,
    DROP CONSTRAINT merch_products_photo_full_width_check,
    DROP CONSTRAINT merch_products_photo_full_size_bytes_check,
    DROP CONSTRAINT merch_products_photo_full_content_type_check,
    DROP CONSTRAINT merch_products_photo_full_object_key_present_check;

ALTER TABLE merch_products
    DROP COLUMN photo_display_height,
    DROP COLUMN photo_display_width,
    DROP COLUMN photo_display_size_bytes,
    DROP COLUMN photo_display_content_type,
    DROP COLUMN photo_display_object_key,
    DROP COLUMN photo_full_height,
    DROP COLUMN photo_full_width;

ALTER TABLE merch_products
    RENAME COLUMN photo_full_size_bytes TO photo_size_bytes;

ALTER TABLE merch_products
    RENAME COLUMN photo_full_content_type TO photo_content_type;

ALTER TABLE merch_products
    RENAME COLUMN photo_full_object_key TO photo_object_key;

ALTER TABLE merch_products
    ADD CONSTRAINT merch_products_photo_object_key_present_check
    CHECK (length(btrim(photo_object_key)) > 0);

ALTER TABLE merch_products
    ADD CONSTRAINT merch_products_photo_content_type_present_check
    CHECK (length(btrim(photo_content_type)) > 0);

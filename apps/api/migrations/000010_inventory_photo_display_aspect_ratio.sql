-- +goose Up
ALTER TABLE merch_products
    DROP CONSTRAINT merch_products_photo_display_aspect_ratio_check;

-- +goose Down
ALTER TABLE merch_products
    ADD CONSTRAINT merch_products_photo_display_aspect_ratio_check
    CHECK (photo_display_width * 3 = photo_display_height * 4);

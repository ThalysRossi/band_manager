-- +goose Up
ALTER TABLE merch_products
    ADD CONSTRAINT merch_products_photo_object_key_present_check
    CHECK (length(btrim(photo_object_key)) > 0);

ALTER TABLE merch_products
    ADD CONSTRAINT merch_products_photo_content_type_present_check
    CHECK (length(btrim(photo_content_type)) > 0);

-- +goose Down
ALTER TABLE merch_products
    DROP CONSTRAINT merch_products_photo_content_type_present_check;

ALTER TABLE merch_products
    DROP CONSTRAINT merch_products_photo_object_key_present_check;

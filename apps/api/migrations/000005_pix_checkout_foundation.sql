-- +goose Up
ALTER TABLE sales
    DROP CONSTRAINT sales_status_check;

ALTER TABLE sales
    ADD CONSTRAINT sales_status_check
    CHECK (status IN ('pending_payment', 'finalized', 'canceled'));

ALTER TABLE sales
    ALTER COLUMN finalized_at DROP NOT NULL;

ALTER TABLE payments
    DROP CONSTRAINT payments_method_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_method_check
    CHECK (method IN ('cash', 'pix'));

ALTER TABLE payments
    DROP CONSTRAINT payments_status_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_status_check
    CHECK (status IN ('provider_pending', 'action_required', 'processing', 'confirmed', 'failed', 'canceled', 'expired'));

ALTER TABLE payments
    ALTER COLUMN confirmed_at DROP NOT NULL;

ALTER TABLE payments
    ADD COLUMN provider TEXT,
    ADD COLUMN provider_order_id TEXT,
    ADD COLUMN provider_payment_id TEXT,
    ADD COLUMN provider_reference_id TEXT,
    ADD COLUMN external_reference TEXT,
    ADD COLUMN provider_status TEXT,
    ADD COLUMN provider_status_detail TEXT,
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN pix_qr_code TEXT,
    ADD COLUMN pix_qr_code_base64 TEXT,
    ADD COLUMN pix_ticket_url TEXT,
    ADD COLUMN raw_provider_response JSONB;

ALTER TABLE inventory_reservations
    DROP CONSTRAINT inventory_reservations_status_check;

ALTER TABLE inventory_reservations
    ADD CONSTRAINT inventory_reservations_status_check
    CHECK (status IN ('reserved', 'consumed', 'released'));

ALTER TABLE inventory_reservations
    ALTER COLUMN consumed_at DROP NOT NULL;

ALTER TABLE inventory_reservations
    ADD COLUMN expires_at TIMESTAMPTZ,
    ADD COLUMN sale_id UUID REFERENCES sales (id);

CREATE INDEX payments_provider_order_id_idx ON payments (provider, provider_order_id);
CREATE INDEX inventory_reservations_expires_at_idx ON inventory_reservations (expires_at) WHERE status = 'reserved';
CREATE INDEX inventory_reservations_sale_id_idx ON inventory_reservations (sale_id);

-- +goose Down
DROP INDEX inventory_reservations_sale_id_idx;
DROP INDEX inventory_reservations_expires_at_idx;
DROP INDEX payments_provider_order_id_idx;

ALTER TABLE inventory_reservations
    DROP COLUMN sale_id,
    DROP COLUMN expires_at;

ALTER TABLE inventory_reservations
    ALTER COLUMN consumed_at SET NOT NULL;

ALTER TABLE inventory_reservations
    DROP CONSTRAINT inventory_reservations_status_check;

ALTER TABLE inventory_reservations
    ADD CONSTRAINT inventory_reservations_status_check
    CHECK (status IN ('consumed'));

ALTER TABLE payments
    DROP COLUMN raw_provider_response,
    DROP COLUMN pix_ticket_url,
    DROP COLUMN pix_qr_code_base64,
    DROP COLUMN pix_qr_code,
    DROP COLUMN expires_at,
    DROP COLUMN provider_status_detail,
    DROP COLUMN provider_status,
    DROP COLUMN external_reference,
    DROP COLUMN provider_reference_id,
    DROP COLUMN provider_payment_id,
    DROP COLUMN provider_order_id,
    DROP COLUMN provider;

ALTER TABLE payments
    ALTER COLUMN confirmed_at SET NOT NULL;

ALTER TABLE payments
    DROP CONSTRAINT payments_status_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_status_check
    CHECK (status IN ('confirmed'));

ALTER TABLE payments
    DROP CONSTRAINT payments_method_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_method_check
    CHECK (method IN ('cash'));

ALTER TABLE sales
    ALTER COLUMN finalized_at SET NOT NULL;

ALTER TABLE sales
    DROP CONSTRAINT sales_status_check;

ALTER TABLE sales
    ADD CONSTRAINT sales_status_check
    CHECK (status IN ('finalized'));

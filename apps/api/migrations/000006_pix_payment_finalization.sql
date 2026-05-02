-- +goose Up
CREATE TABLE payment_events (
    id UUID PRIMARY KEY,
    provider TEXT NOT NULL,
    provider_order_id TEXT,
    band_id UUID REFERENCES bands (id),
    sale_id UUID REFERENCES sales (id),
    payment_id UUID REFERENCES payments (id),
    webhook_request_id TEXT,
    signature_timestamp TIMESTAMPTZ,
    signature_verified BOOLEAN NOT NULL,
    raw_query TEXT NOT NULL,
    raw_body TEXT NOT NULL,
    processing_status TEXT NOT NULL CHECK (processing_status IN ('processed', 'rejected', 'failed')),
    processing_error TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX payment_events_provider_webhook_request_id_idx
    ON payment_events (provider, webhook_request_id)
    WHERE webhook_request_id IS NOT NULL;

CREATE INDEX payment_events_provider_order_id_idx ON payment_events (provider, provider_order_id);
CREATE INDEX payment_events_payment_id_idx ON payment_events (payment_id);

-- +goose Down
DROP INDEX payment_events_payment_id_idx;
DROP INDEX payment_events_provider_order_id_idx;
DROP INDEX payment_events_provider_webhook_request_id_idx;
DROP TABLE payment_events;

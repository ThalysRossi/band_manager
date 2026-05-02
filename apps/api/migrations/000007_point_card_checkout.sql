-- +goose Up
ALTER TABLE payments
    DROP CONSTRAINT payments_method_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_method_check
    CHECK (method IN ('cash', 'pix', 'card'));

ALTER TABLE payments
    ADD COLUMN point_terminal_id TEXT,
    ADD COLUMN card_payment_type TEXT CHECK (card_payment_type IN ('credit_card', 'debit_card')),
    ADD COLUMN card_installments INTEGER CHECK (card_installments IS NULL OR card_installments > 0);

CREATE INDEX payments_point_terminal_id_idx ON payments (point_terminal_id) WHERE point_terminal_id IS NOT NULL;

-- +goose Down
DROP INDEX payments_point_terminal_id_idx;

ALTER TABLE payments
    DROP COLUMN card_installments,
    DROP COLUMN card_payment_type,
    DROP COLUMN point_terminal_id;

ALTER TABLE payments
    DROP CONSTRAINT payments_method_check;

ALTER TABLE payments
    ADD CONSTRAINT payments_method_check
    CHECK (method IN ('cash', 'pix'));

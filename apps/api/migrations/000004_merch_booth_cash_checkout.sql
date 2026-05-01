-- +goose Up
ALTER TABLE inventory_movements
    DROP CONSTRAINT inventory_movements_movement_type_check;

ALTER TABLE inventory_movements
    ADD CONSTRAINT inventory_movements_movement_type_check
    CHECK (movement_type IN ('initial_stock', 'manual_adjustment', 'sale'));

CREATE TABLE inventory_reservations (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    product_id UUID NOT NULL REFERENCES merch_products (id),
    variant_id UUID NOT NULL REFERENCES merch_variants (id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status TEXT NOT NULL CHECK (status IN ('consumed')),
    created_by_user_id UUID NOT NULL REFERENCES users (id),
    consumed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE sales (
    id UUID PRIMARY KEY,
    band_id UUID NOT NULL REFERENCES bands (id),
    created_by_user_id UUID NOT NULL REFERENCES users (id),
    status TEXT NOT NULL CHECK (status IN ('finalized')),
    total_amount INTEGER NOT NULL CHECK (total_amount >= 0),
    expected_profit_amount INTEGER NOT NULL,
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    finalized_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE sale_items (
    id UUID PRIMARY KEY,
    sale_id UUID NOT NULL REFERENCES sales (id),
    band_id UUID NOT NULL REFERENCES bands (id),
    product_id UUID NOT NULL REFERENCES merch_products (id),
    variant_id UUID NOT NULL REFERENCES merch_variants (id),
    product_name TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('shirt', 'hoodie', 'tote_bag', 'patch', 'sticker', 'vinyl', 'cd', 'cassette', 'accessory')),
    size TEXT NOT NULL CHECK (size IN ('not_applicable', 'one_size', 'pp', 'p', 'm', 'g', 'gg', 'xgg')),
    colour TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price_amount INTEGER NOT NULL CHECK (unit_price_amount >= 0),
    unit_cost_amount INTEGER NOT NULL CHECK (unit_cost_amount >= 0),
    line_total_amount INTEGER NOT NULL CHECK (line_total_amount >= 0),
    expected_profit_amount INTEGER NOT NULL,
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE payments (
    id UUID PRIMARY KEY,
    sale_id UUID NOT NULL REFERENCES sales (id),
    band_id UUID NOT NULL REFERENCES bands (id),
    method TEXT NOT NULL CHECK (method IN ('cash')),
    status TEXT NOT NULL CHECK (status IN ('confirmed')),
    amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    confirmed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    sale_id UUID NOT NULL REFERENCES sales (id),
    sale_item_id UUID NOT NULL REFERENCES sale_items (id),
    band_id UUID NOT NULL REFERENCES bands (id),
    transaction_type TEXT NOT NULL CHECK (transaction_type = 'sale_item'),
    amount_minor INTEGER NOT NULL CHECK (amount_minor >= 0),
    currency TEXT NOT NULL CHECK (currency = 'BRL'),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX inventory_reservations_band_created_at_idx ON inventory_reservations (band_id, created_at);
CREATE INDEX sales_band_created_at_idx ON sales (band_id, created_at);
CREATE INDEX sale_items_sale_id_idx ON sale_items (sale_id);
CREATE INDEX payments_sale_id_idx ON payments (sale_id);
CREATE INDEX transactions_sale_id_idx ON transactions (sale_id);

-- +goose Down
DROP TABLE transactions;
DROP TABLE payments;
DROP TABLE sale_items;
DROP TABLE sales;
DROP TABLE inventory_reservations;

ALTER TABLE inventory_movements
    DROP CONSTRAINT inventory_movements_movement_type_check;

ALTER TABLE inventory_movements
    ADD CONSTRAINT inventory_movements_movement_type_check
    CHECK (movement_type IN ('initial_stock', 'manual_adjustment'));

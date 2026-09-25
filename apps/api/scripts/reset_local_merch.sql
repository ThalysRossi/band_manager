BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM payments
        WHERE status IN ('provider_pending', 'action_required', 'processing')
    ) THEN
        RAISE EXCEPTION 'local merch reset refused: a provider payment is pending';
    END IF;
END $$;

TRUNCATE TABLE payment_events, transactions, sale_items, inventory_reservations,
    payments, sales, inventory_movements, merch_variants, merch_products;

DELETE FROM idempotency_records
WHERE operation LIKE 'merch_booth_%' OR operation LIKE 'inventory_%';

DELETE FROM audit_logs
WHERE action LIKE 'inventory.%' OR action LIKE 'merch_booth.%';

COMMIT;

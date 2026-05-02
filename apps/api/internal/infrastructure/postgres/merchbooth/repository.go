package merchbooth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
)

const cashCheckoutOperation = "merch_booth_cash_checkout"
const pixCheckoutOperation = "merch_booth_pix_checkout"

type Repository struct {
	pool *pgxpool.Pool
}

type boothVariantRow struct {
	ProductID   string
	VariantID   string
	ProductName string
	Category    string
	Size        string
	Colour      string
	PriceAmount int
	CostAmount  int
	Currency    string
	Quantity    int
	PhotoKey    string
	PhotoType   string
	PhotoSize   int
}

type checkoutLine struct {
	CartItem      applicationmerchbooth.CartItem
	Variant       applicationmerchbooth.BoothItem
	QuantityAfter int
	LineTotal     int
	LineProfit    int
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (repository Repository) ListBoothItems(ctx context.Context, query applicationmerchbooth.ListBoothItemsQuery) ([]applicationmerchbooth.BoothItem, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT merch_products.id,
			merch_variants.id,
			merch_products.name,
			merch_products.category,
			merch_variants.size,
			merch_variants.colour,
			merch_variants.price_amount,
			merch_variants.cost_amount,
			merch_variants.currency,
			merch_variants.quantity,
			merch_products.photo_object_key,
			merch_products.photo_content_type,
			merch_products.photo_size_bytes
		FROM merch_variants
		INNER JOIN merch_products ON merch_products.id = merch_variants.product_id
		WHERE merch_variants.band_id = $1
			AND merch_variants.deleted_at IS NULL
			AND merch_products.deleted_at IS NULL
		ORDER BY merch_products.created_at ASC, merch_variants.created_at ASC, merch_variants.id ASC
	`, query.Account.BandID)
	if err != nil {
		return nil, fmt.Errorf("query merch booth items band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	items := make([]applicationmerchbooth.BoothItem, 0)
	for rows.Next() {
		item, err := scanBoothItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan merch booth item band_id=%q: %w", query.Account.BandID, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate merch booth items band_id=%q: %w", query.Account.BandID, err)
	}

	return items, nil
}

func (repository Repository) CreateCashCheckout(ctx context.Context, command applicationmerchbooth.CreateCashCheckoutCommand) (applicationmerchbooth.Sale, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("begin cash checkout transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	requestHash, err := hashCashCheckoutRequest(command)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}

	existingSale, found, err := findIdempotentCashCheckout(ctx, tx, command.Account.BandID, command.IdempotencyKey, requestHash)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}
	if found {
		return existingSale, nil
	}

	variantRows, err := lockCheckoutVariants(ctx, tx, command)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}

	variantsByID := make(map[string]applicationmerchbooth.BoothItem, len(variantRows))
	for _, variant := range variantRows {
		variantsByID[variant.VariantID] = variant
	}

	saleID := uuid.NewString()
	paymentID := uuid.NewString()
	lines := make([]checkoutLine, 0, len(command.Items))
	saleItems := make([]applicationmerchbooth.SaleItem, 0, len(command.Items))
	transactions := make([]applicationmerchbooth.Transaction, 0, len(command.Items))
	totalAmount := 0
	expectedProfitAmount := 0

	for _, item := range command.Items {
		variant, ok := variantsByID[item.VariantID]
		if !ok {
			return applicationmerchbooth.Sale{}, fmt.Errorf("%w: band_id=%q variant_id=%q", applicationmerchbooth.ErrBoothItemNotFound, command.Account.BandID, item.VariantID)
		}

		reservedQuantity, err := activeReservedQuantity(ctx, tx, command.Account.BandID, item.VariantID, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, err
		}
		availableQuantity := variant.Quantity - reservedQuantity
		if availableQuantity < item.Quantity {
			return applicationmerchbooth.Sale{}, fmt.Errorf("%w: band_id=%q variant_id=%q requested=%d available=%d", applicationmerchbooth.ErrInsufficientStock, command.Account.BandID, item.VariantID, item.Quantity, availableQuantity)
		}

		quantityAfter := variant.Quantity - item.Quantity
		lineTotal := variant.Price.Amount * item.Quantity
		lineCost := variant.Cost.Amount * item.Quantity
		lineProfit := lineTotal - lineCost
		totalAmount += lineTotal
		expectedProfitAmount += lineProfit

		lines = append(lines, checkoutLine{
			CartItem:      item,
			Variant:       variant,
			QuantityAfter: quantityAfter,
			LineTotal:     lineTotal,
			LineProfit:    lineProfit,
		})
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO sales (
			id, band_id, created_by_user_id, status, total_amount,
			expected_profit_amount, currency, finalized_at, idempotency_key,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $8, $8)
	`, saleID, command.Account.BandID, command.Account.UserID, applicationmerchbooth.SaleStatusFinalized, totalAmount, expectedProfitAmount, "BRL", command.CreatedAt, command.IdempotencyKey)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("insert cash checkout sale band_id=%q sale_id=%q: %w", command.Account.BandID, saleID, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO payments (
			id, sale_id, band_id, method, status, amount_minor,
			currency, confirmed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $8)
	`, paymentID, saleID, command.Account.BandID, applicationmerchbooth.PaymentMethodCash, applicationmerchbooth.PaymentStatusConfirmed, totalAmount, "BRL", command.CreatedAt)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("insert cash checkout payment band_id=%q sale_id=%q: %w", command.Account.BandID, saleID, err)
	}

	for _, line := range lines {
		item := line.CartItem
		variant := line.Variant
		reservationID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO inventory_reservations (
				id, sale_id, band_id, product_id, variant_id, quantity, status,
				created_by_user_id, consumed_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $9)
		`, reservationID, saleID, command.Account.BandID, variant.ProductID, variant.VariantID, item.Quantity, "consumed", command.Account.UserID, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, fmt.Errorf("insert cash checkout reservation band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE merch_variants
			SET quantity = $1, updated_at = $2
			WHERE id = $3 AND band_id = $4 AND deleted_at IS NULL
		`, line.QuantityAfter, command.CreatedAt, item.VariantID, command.Account.BandID)
		if err != nil {
			return applicationmerchbooth.Sale{}, fmt.Errorf("decrement checkout inventory band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO inventory_movements (
				id, band_id, product_id, variant_id, movement_type,
				quantity_delta, quantity_after, reason, actor_user_id, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, uuid.NewString(), command.Account.BandID, variant.ProductID, variant.VariantID, "sale", -item.Quantity, line.QuantityAfter, "merch_booth.cash_checkout", command.Account.UserID, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, fmt.Errorf("insert checkout inventory movement band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}

		saleItemID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (
				id, sale_id, band_id, product_id, variant_id, product_name,
				category, size, colour, quantity, unit_price_amount, unit_cost_amount,
				line_total_amount, expected_profit_amount, currency, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`, saleItemID, saleID, command.Account.BandID, variant.ProductID, variant.VariantID, variant.ProductName, variant.Category, variant.Size, variant.Colour, item.Quantity, variant.Price.Amount, variant.Cost.Amount, line.LineTotal, line.LineProfit, variant.Price.Currency, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, fmt.Errorf("insert cash checkout sale item band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}

		transactionID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO transactions (
				id, sale_id, sale_item_id, band_id, transaction_type,
				amount_minor, currency, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, transactionID, saleID, saleItemID, command.Account.BandID, "sale_item", line.LineTotal, variant.Price.Currency, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, fmt.Errorf("insert cash checkout transaction band_id=%q sale_item_id=%q: %w", command.Account.BandID, saleItemID, err)
		}

		saleItems = append(saleItems, applicationmerchbooth.SaleItem{
			ID:             saleItemID,
			SaleID:         saleID,
			ProductID:      variant.ProductID,
			VariantID:      variant.VariantID,
			ProductName:    variant.ProductName,
			Category:       variant.Category,
			Size:           variant.Size,
			Colour:         variant.Colour,
			Quantity:       item.Quantity,
			UnitPrice:      variant.Price,
			UnitCost:       variant.Cost,
			LineTotal:      inventorydomain.Money{Amount: line.LineTotal, Currency: variant.Price.Currency},
			ExpectedProfit: inventorydomain.Money{Amount: line.LineProfit, Currency: variant.Price.Currency},
			CreatedAt:      command.CreatedAt,
		})
		transactions = append(transactions, applicationmerchbooth.Transaction{
			ID:         transactionID,
			SaleID:     saleID,
			SaleItemID: saleItemID,
			Amount:     inventorydomain.Money{Amount: line.LineTotal, Currency: variant.Price.Currency},
			CreatedAt:  command.CreatedAt,
		})
	}

	sale := applicationmerchbooth.Sale{
		ID:             saleID,
		BandID:         command.Account.BandID,
		Status:         applicationmerchbooth.SaleStatusFinalized,
		Total:          inventorydomain.Money{Amount: totalAmount, Currency: "BRL"},
		ExpectedProfit: inventorydomain.Money{Amount: expectedProfitAmount, Currency: "BRL"},
		Items:          saleItems,
		Payment: applicationmerchbooth.Payment{
			ID:        paymentID,
			SaleID:    saleID,
			Method:    applicationmerchbooth.PaymentMethodCash,
			Status:    applicationmerchbooth.PaymentStatusConfirmed,
			Amount:    inventorydomain.Money{Amount: totalAmount, Currency: "BRL"},
			CreatedAt: command.CreatedAt,
			UpdatedAt: command.CreatedAt,
		},
		Transactions: transactions,
		CreatedAt:    command.CreatedAt,
		UpdatedAt:    command.CreatedAt,
	}

	responseBody, err := json.Marshal(sale)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("marshal idempotent cash checkout response band_id=%q sale_id=%q: %w", command.Account.BandID, saleID, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_records (
			id, scope_id, band_id, operation, idempotency_key,
			request_hash, response_body, status_code, expires_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, uuid.NewString(), command.Account.BandID, command.Account.BandID, cashCheckoutOperation, command.IdempotencyKey, requestHash, responseBody, 201, command.CreatedAt.Add(time.Hour), command.CreatedAt)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("insert cash checkout idempotency record band_id=%q key=%q: %w", command.Account.BandID, command.IdempotencyKey, err)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "merch_booth.cash_checkout_finalized", "sale", saleID, command.RequestID, command.IdempotencyKey, command.CreatedAt); err != nil {
		return applicationmerchbooth.Sale{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("commit cash checkout transaction band_id=%q sale_id=%q: %w", command.Account.BandID, saleID, err)
	}

	return sale, nil
}

func (repository Repository) ReservePixCheckout(ctx context.Context, command applicationmerchbooth.CreatePixCheckoutCommand) (applicationmerchbooth.Sale, bool, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("begin pix checkout reservation transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	requestHash, err := applicationmerchbooth.HashPixCheckoutRequest(command)
	if err != nil {
		return applicationmerchbooth.Sale{}, false, err
	}

	existingSale, found, err := findIdempotentPixCheckout(ctx, tx, command.Account.BandID, command.IdempotencyKey, requestHash)
	if err != nil {
		return applicationmerchbooth.Sale{}, false, err
	}
	if found {
		return existingSale, true, nil
	}

	variantRows, err := lockCheckoutVariants(ctx, tx, applicationmerchbooth.CreateCashCheckoutCommand{
		Account:        command.Account,
		Items:          command.Items,
		IdempotencyKey: command.IdempotencyKey,
		RequestID:      command.RequestID,
		CreatedAt:      command.CreatedAt,
	})
	if err != nil {
		return applicationmerchbooth.Sale{}, false, err
	}

	variantsByID := make(map[string]applicationmerchbooth.BoothItem, len(variantRows))
	for _, variant := range variantRows {
		variantsByID[variant.VariantID] = variant
	}

	lines := make([]checkoutLine, 0, len(command.Items))
	totalAmount := 0
	expectedProfitAmount := 0

	for _, item := range command.Items {
		variant, ok := variantsByID[item.VariantID]
		if !ok {
			return applicationmerchbooth.Sale{}, false, fmt.Errorf("%w: band_id=%q variant_id=%q", applicationmerchbooth.ErrBoothItemNotFound, command.Account.BandID, item.VariantID)
		}
		reservedQuantity, err := activeReservedQuantity(ctx, tx, command.Account.BandID, item.VariantID, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, false, err
		}
		availableQuantity := variant.Quantity - reservedQuantity
		if availableQuantity < item.Quantity {
			return applicationmerchbooth.Sale{}, false, fmt.Errorf("%w: band_id=%q variant_id=%q requested=%d available=%d", applicationmerchbooth.ErrInsufficientStock, command.Account.BandID, item.VariantID, item.Quantity, availableQuantity)
		}

		lineTotal := variant.Price.Amount * item.Quantity
		lineCost := variant.Cost.Amount * item.Quantity
		lineProfit := lineTotal - lineCost
		totalAmount += lineTotal
		expectedProfitAmount += lineProfit

		lines = append(lines, checkoutLine{
			CartItem:   item,
			Variant:    variant,
			LineTotal:  lineTotal,
			LineProfit: lineProfit,
		})
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO sales (
			id, band_id, created_by_user_id, status, total_amount,
			expected_profit_amount, currency, finalized_at, idempotency_key,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9, $9)
	`, command.SaleID, command.Account.BandID, command.Account.UserID, applicationmerchbooth.SaleStatusPendingPayment, totalAmount, expectedProfitAmount, "BRL", command.IdempotencyKey, command.CreatedAt)
	if err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("insert pix checkout sale band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO payments (
			id, sale_id, band_id, method, status, amount_minor,
			currency, confirmed_at, external_reference, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8, $9, $10, $10)
	`, command.PaymentID, command.SaleID, command.Account.BandID, applicationmerchbooth.PaymentMethodPix, applicationmerchbooth.PaymentStatusProviderPending, totalAmount, "BRL", command.ExternalReference, command.ExpiresAt, command.CreatedAt)
	if err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("insert pix checkout payment band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	saleItems := make([]applicationmerchbooth.SaleItem, 0, len(command.Items))
	for _, line := range lines {
		item := line.CartItem
		variant := line.Variant
		_, err = tx.Exec(ctx, `
			INSERT INTO inventory_reservations (
				id, sale_id, band_id, product_id, variant_id, quantity, status,
				created_by_user_id, consumed_at, expires_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, $9, $10, $10)
		`, uuid.NewString(), command.SaleID, command.Account.BandID, variant.ProductID, variant.VariantID, item.Quantity, "reserved", command.Account.UserID, command.ExpiresAt, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, false, fmt.Errorf("insert pix checkout reservation band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}

		saleItemID := uuid.NewString()
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (
				id, sale_id, band_id, product_id, variant_id, product_name,
				category, size, colour, quantity, unit_price_amount, unit_cost_amount,
				line_total_amount, expected_profit_amount, currency, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`, saleItemID, command.SaleID, command.Account.BandID, variant.ProductID, variant.VariantID, variant.ProductName, variant.Category, variant.Size, variant.Colour, item.Quantity, variant.Price.Amount, variant.Cost.Amount, line.LineTotal, line.LineProfit, variant.Price.Currency, command.CreatedAt)
		if err != nil {
			return applicationmerchbooth.Sale{}, false, fmt.Errorf("insert pix checkout sale item band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}

		saleItems = append(saleItems, applicationmerchbooth.SaleItem{
			ID:             saleItemID,
			SaleID:         command.SaleID,
			ProductID:      variant.ProductID,
			VariantID:      variant.VariantID,
			ProductName:    variant.ProductName,
			Category:       variant.Category,
			Size:           variant.Size,
			Colour:         variant.Colour,
			Quantity:       item.Quantity,
			UnitPrice:      variant.Price,
			UnitCost:       variant.Cost,
			LineTotal:      inventorydomain.Money{Amount: line.LineTotal, Currency: variant.Price.Currency},
			ExpectedProfit: inventorydomain.Money{Amount: line.LineProfit, Currency: variant.Price.Currency},
			CreatedAt:      command.CreatedAt,
		})
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "merch_booth.pix_checkout_reserved", "sale", command.SaleID, command.RequestID, command.IdempotencyKey, command.CreatedAt); err != nil {
		return applicationmerchbooth.Sale{}, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("commit pix checkout reservation transaction band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	return applicationmerchbooth.Sale{
		ID:             command.SaleID,
		BandID:         command.Account.BandID,
		Status:         applicationmerchbooth.SaleStatusPendingPayment,
		Total:          inventorydomain.Money{Amount: totalAmount, Currency: "BRL"},
		ExpectedProfit: inventorydomain.Money{Amount: expectedProfitAmount, Currency: "BRL"},
		Items:          saleItems,
		Payment: applicationmerchbooth.Payment{
			ID:                command.PaymentID,
			SaleID:            command.SaleID,
			Method:            applicationmerchbooth.PaymentMethodPix,
			Status:            applicationmerchbooth.PaymentStatusProviderPending,
			Amount:            inventorydomain.Money{Amount: totalAmount, Currency: "BRL"},
			ExternalReference: command.ExternalReference,
			ExpiresAt:         command.ExpiresAt,
			CreatedAt:         command.CreatedAt,
			UpdatedAt:         command.CreatedAt,
		},
		Transactions: []applicationmerchbooth.Transaction{},
		CreatedAt:    command.CreatedAt,
		UpdatedAt:    command.CreatedAt,
	}, false, nil
}

func (repository Repository) CompletePixCheckoutPayment(ctx context.Context, command applicationmerchbooth.CompletePixCheckoutPaymentCommand) (applicationmerchbooth.Sale, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("begin pix checkout completion transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rawProviderResponse := command.ProviderResult.RawProviderResponse
	if len(rawProviderResponse) == 0 {
		return applicationmerchbooth.Sale{}, fmt.Errorf("pix provider response is required band_id=%q sale_id=%q", command.Account.BandID, command.SaleID)
	}

	_, err = tx.Exec(ctx, `
		UPDATE payments
		SET status = $1,
			provider = $2,
			provider_order_id = $3,
			provider_payment_id = $4,
			provider_reference_id = $5,
			external_reference = $6,
			provider_status = $7,
			provider_status_detail = $8,
			expires_at = $9,
			pix_qr_code = $10,
			pix_qr_code_base64 = $11,
			pix_ticket_url = $12,
			raw_provider_response = $13,
			updated_at = $14
		WHERE id = $15 AND sale_id = $16 AND band_id = $17
	`, command.ProviderResult.LocalStatus, command.ProviderResult.Provider, command.ProviderResult.ProviderOrderID, command.ProviderResult.ProviderPaymentID, command.ProviderResult.ProviderReferenceID, command.ProviderResult.ExternalReference, command.ProviderResult.ProviderStatus, command.ProviderResult.ProviderStatusDetail, command.ProviderResult.ExpiresAt, command.ProviderResult.QRCode, command.ProviderResult.QRCodeBase64, command.ProviderResult.TicketURL, rawProviderResponse, command.UpdatedAt, command.PaymentID, command.SaleID, command.Account.BandID)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("update pix checkout payment band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	sale, err := getSaleByID(ctx, tx, command.Account.BandID, command.SaleID)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}

	responseBody, err := json.Marshal(sale)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("marshal idempotent pix checkout response band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_records (
			id, scope_id, band_id, operation, idempotency_key,
			request_hash, response_body, status_code, expires_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, uuid.NewString(), command.Account.BandID, command.Account.BandID, pixCheckoutOperation, command.IdempotencyKey, command.RequestHash, responseBody, 201, command.UpdatedAt.Add(time.Hour), command.UpdatedAt)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("insert pix checkout idempotency record band_id=%q key=%q: %w", command.Account.BandID, command.IdempotencyKey, err)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "merch_booth.pix_checkout_payment_created", "payment", command.PaymentID, command.RequestID, command.IdempotencyKey, command.UpdatedAt); err != nil {
		return applicationmerchbooth.Sale{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("commit pix checkout completion transaction band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	return sale, nil
}

func (repository Repository) FailPixCheckoutPaymentCreation(ctx context.Context, command applicationmerchbooth.FailPixCheckoutPaymentCreationCommand) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin pix checkout failure transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE inventory_reservations
		SET status = $1, updated_at = $2
		WHERE band_id = $3 AND sale_id = $4 AND status = $5
	`, "released", command.UpdatedAt, command.Account.BandID, command.SaleID, "reserved")
	if err != nil {
		return fmt.Errorf("release failed pix checkout reservations band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE payments
		SET status = $1, updated_at = $2
		WHERE id = $3 AND sale_id = $4 AND band_id = $5
	`, applicationmerchbooth.PaymentStatusFailed, command.UpdatedAt, command.PaymentID, command.SaleID, command.Account.BandID)
	if err != nil {
		return fmt.Errorf("mark failed pix checkout payment band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE sales
		SET status = $1, updated_at = $2
		WHERE id = $3 AND band_id = $4
	`, applicationmerchbooth.SaleStatusCanceled, command.UpdatedAt, command.SaleID, command.Account.BandID)
	if err != nil {
		return fmt.Errorf("mark failed pix checkout sale canceled band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	if err := insertAuditLog(ctx, tx, command.Account.UserID, command.Account.BandID, "merch_booth.pix_checkout_payment_creation_failed", "sale", command.SaleID, command.RequestID, command.IdempotencyKey, command.UpdatedAt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit pix checkout failure transaction band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
	}

	return nil
}

func lockCheckoutVariants(ctx context.Context, tx pgx.Tx, command applicationmerchbooth.CreateCashCheckoutCommand) ([]applicationmerchbooth.BoothItem, error) {
	variants := make([]applicationmerchbooth.BoothItem, 0, len(command.Items))
	for _, item := range command.Items {
		row := tx.QueryRow(ctx, `
			SELECT merch_products.id,
				merch_variants.id,
				merch_products.name,
				merch_products.category,
				merch_variants.size,
				merch_variants.colour,
				merch_variants.price_amount,
				merch_variants.cost_amount,
				merch_variants.currency,
				merch_variants.quantity,
				merch_products.photo_object_key,
				merch_products.photo_content_type,
				merch_products.photo_size_bytes
			FROM merch_variants
			INNER JOIN merch_products ON merch_products.id = merch_variants.product_id
			WHERE merch_variants.band_id = $1
				AND merch_variants.id = $2
				AND merch_variants.deleted_at IS NULL
				AND merch_products.deleted_at IS NULL
			FOR UPDATE OF merch_variants
		`, command.Account.BandID, item.VariantID)

		variant, err := scanBoothItem(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: band_id=%q variant_id=%q", applicationmerchbooth.ErrBoothItemNotFound, command.Account.BandID, item.VariantID)
		}
		if err != nil {
			return nil, fmt.Errorf("lock checkout variant band_id=%q variant_id=%q: %w", command.Account.BandID, item.VariantID, err)
		}
		variants = append(variants, variant)
	}

	return variants, nil
}

func activeReservedQuantity(ctx context.Context, tx pgx.Tx, bandID string, variantID string, at time.Time) (int, error) {
	var reservedQuantity int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity), 0)
		FROM inventory_reservations
		WHERE band_id = $1
			AND variant_id = $2
			AND status = $3
			AND (expires_at IS NULL OR expires_at > $4)
	`, bandID, variantID, "reserved", at).Scan(&reservedQuantity)
	if err != nil {
		return 0, fmt.Errorf("query active reserved quantity band_id=%q variant_id=%q: %w", bandID, variantID, err)
	}

	return reservedQuantity, nil
}

func findIdempotentCashCheckout(ctx context.Context, tx pgx.Tx, bandID string, idempotencyKey string, requestHash string) (applicationmerchbooth.Sale, bool, error) {
	var storedRequestHash string
	var responseBody []byte
	err := tx.QueryRow(ctx, `
		SELECT request_hash, response_body
		FROM idempotency_records
		WHERE scope_id = $1 AND operation = $2 AND idempotency_key = $3 AND expires_at > NOW()
	`, bandID, cashCheckoutOperation, idempotencyKey).Scan(&storedRequestHash, &responseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return applicationmerchbooth.Sale{}, false, nil
	}
	if err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("query cash checkout idempotency record band_id=%q key=%q: %w", bandID, idempotencyKey, err)
	}

	if storedRequestHash != requestHash {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("%w: band_id=%q key=%q", applicationmerchbooth.ErrIdempotencyConflict, bandID, idempotencyKey)
	}

	var sale applicationmerchbooth.Sale
	if err := json.Unmarshal(responseBody, &sale); err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("parse idempotent cash checkout response band_id=%q key=%q: %w", bandID, idempotencyKey, err)
	}

	return sale, true, nil
}

func findIdempotentPixCheckout(ctx context.Context, tx pgx.Tx, bandID string, idempotencyKey string, requestHash string) (applicationmerchbooth.Sale, bool, error) {
	var storedRequestHash string
	var responseBody []byte
	err := tx.QueryRow(ctx, `
		SELECT request_hash, response_body
		FROM idempotency_records
		WHERE scope_id = $1 AND operation = $2 AND idempotency_key = $3 AND expires_at > NOW()
	`, bandID, pixCheckoutOperation, idempotencyKey).Scan(&storedRequestHash, &responseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return applicationmerchbooth.Sale{}, false, nil
	}
	if err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("query pix checkout idempotency record band_id=%q key=%q: %w", bandID, idempotencyKey, err)
	}

	if storedRequestHash != requestHash {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("%w: band_id=%q key=%q", applicationmerchbooth.ErrIdempotencyConflict, bandID, idempotencyKey)
	}

	var sale applicationmerchbooth.Sale
	if err := json.Unmarshal(responseBody, &sale); err != nil {
		return applicationmerchbooth.Sale{}, false, fmt.Errorf("parse idempotent pix checkout response band_id=%q key=%q: %w", bandID, idempotencyKey, err)
	}

	return sale, true, nil
}

func getSaleByID(ctx context.Context, tx pgx.Tx, bandID string, saleID string) (applicationmerchbooth.Sale, error) {
	var sale applicationmerchbooth.Sale
	var saleStatus string
	var currency string
	err := tx.QueryRow(ctx, `
		SELECT id, band_id, status, total_amount, expected_profit_amount, currency, created_at, updated_at
		FROM sales
		WHERE id = $1 AND band_id = $2
	`, saleID, bandID).Scan(&sale.ID, &sale.BandID, &saleStatus, &sale.Total.Amount, &sale.ExpectedProfit.Amount, &currency, &sale.CreatedAt, &sale.UpdatedAt)
	if err != nil {
		return applicationmerchbooth.Sale{}, fmt.Errorf("query sale band_id=%q sale_id=%q: %w", bandID, saleID, err)
	}
	sale.Status = applicationmerchbooth.SaleStatus(saleStatus)
	sale.Total.Currency = currency
	sale.ExpectedProfit.Currency = currency

	payment, err := getSalePayment(ctx, tx, bandID, saleID)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}
	items, err := getSaleItems(ctx, tx, bandID, saleID)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}
	transactions, err := getSaleTransactions(ctx, tx, bandID, saleID)
	if err != nil {
		return applicationmerchbooth.Sale{}, err
	}

	sale.Payment = payment
	sale.Items = items
	sale.Transactions = transactions

	return sale, nil
}

func hashCashCheckoutRequest(command applicationmerchbooth.CreateCashCheckoutCommand) (string, error) {
	body, err := json.Marshal(struct {
		BandID string                              `json:"bandId"`
		Items  []applicationmerchbooth.CartItem    `json:"items"`
		Method applicationmerchbooth.PaymentMethod `json:"method"`
	}{
		BandID: command.Account.BandID,
		Items:  command.Items,
		Method: applicationmerchbooth.PaymentMethodCash,
	})
	if err != nil {
		return "", fmt.Errorf("marshal cash checkout request hash body: %w", err)
	}

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func getSalePayment(ctx context.Context, tx pgx.Tx, bandID string, saleID string) (applicationmerchbooth.Payment, error) {
	var payment applicationmerchbooth.Payment
	var method string
	var status string
	var currency string
	var provider sql.NullString
	var providerOrderID sql.NullString
	var providerPaymentID sql.NullString
	var providerReferenceID sql.NullString
	var externalReference sql.NullString
	var providerStatus sql.NullString
	var providerStatusDetail sql.NullString
	var expiresAt sql.NullTime
	var pixQRCode sql.NullString
	var pixQRCodeBase64 sql.NullString
	var pixTicketURL sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT id, sale_id, method, status, amount_minor, currency,
			provider, provider_order_id, provider_payment_id, provider_reference_id,
			external_reference, provider_status, provider_status_detail, expires_at,
			pix_qr_code, pix_qr_code_base64, pix_ticket_url, created_at, updated_at
		FROM payments
		WHERE sale_id = $1 AND band_id = $2
	`, saleID, bandID).Scan(
		&payment.ID,
		&payment.SaleID,
		&method,
		&status,
		&payment.Amount.Amount,
		&currency,
		&provider,
		&providerOrderID,
		&providerPaymentID,
		&providerReferenceID,
		&externalReference,
		&providerStatus,
		&providerStatusDetail,
		&expiresAt,
		&pixQRCode,
		&pixQRCodeBase64,
		&pixTicketURL,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return applicationmerchbooth.Payment{}, fmt.Errorf("query sale payment band_id=%q sale_id=%q: %w", bandID, saleID, err)
	}

	payment.Method = applicationmerchbooth.PaymentMethod(method)
	payment.Status = applicationmerchbooth.PaymentStatus(status)
	payment.Amount.Currency = currency
	payment.Provider = provider.String
	payment.ProviderOrderID = providerOrderID.String
	payment.ProviderPaymentID = providerPaymentID.String
	payment.ProviderReferenceID = providerReferenceID.String
	payment.ExternalReference = externalReference.String
	payment.ProviderStatus = providerStatus.String
	payment.ProviderStatusDetail = providerStatusDetail.String
	if expiresAt.Valid {
		payment.ExpiresAt = expiresAt.Time
	}
	payment.PixQRCode = pixQRCode.String
	payment.PixQRCodeBase64 = pixQRCodeBase64.String
	payment.PixTicketURL = pixTicketURL.String

	return payment, nil
}

func getSaleItems(ctx context.Context, tx pgx.Tx, bandID string, saleID string) ([]applicationmerchbooth.SaleItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, sale_id, product_id, variant_id, product_name, category, size, colour,
			quantity, unit_price_amount, unit_cost_amount, line_total_amount,
			expected_profit_amount, currency, created_at
		FROM sale_items
		WHERE sale_id = $1 AND band_id = $2
		ORDER BY created_at ASC, id ASC
	`, saleID, bandID)
	if err != nil {
		return nil, fmt.Errorf("query sale items band_id=%q sale_id=%q: %w", bandID, saleID, err)
	}
	defer rows.Close()

	items := make([]applicationmerchbooth.SaleItem, 0)
	for rows.Next() {
		var item applicationmerchbooth.SaleItem
		var category string
		var size string
		var currency string
		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.ProductID,
			&item.VariantID,
			&item.ProductName,
			&category,
			&size,
			&item.Colour,
			&item.Quantity,
			&item.UnitPrice.Amount,
			&item.UnitCost.Amount,
			&item.LineTotal.Amount,
			&item.ExpectedProfit.Amount,
			&currency,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sale item band_id=%q sale_id=%q: %w", bandID, saleID, err)
		}
		parsedCategory, err := inventorydomain.ParseCategory(category)
		if err != nil {
			return nil, err
		}
		parsedSize, err := inventorydomain.ParseSize(size)
		if err != nil {
			return nil, err
		}
		item.Category = parsedCategory
		item.Size = parsedSize
		item.UnitPrice.Currency = currency
		item.UnitCost.Currency = currency
		item.LineTotal.Currency = currency
		item.ExpectedProfit.Currency = currency
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sale items band_id=%q sale_id=%q: %w", bandID, saleID, err)
	}

	return items, nil
}

func getSaleTransactions(ctx context.Context, tx pgx.Tx, bandID string, saleID string) ([]applicationmerchbooth.Transaction, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, sale_id, sale_item_id, amount_minor, currency, created_at
		FROM transactions
		WHERE sale_id = $1 AND band_id = $2
		ORDER BY created_at ASC, id ASC
	`, saleID, bandID)
	if err != nil {
		return nil, fmt.Errorf("query sale transactions band_id=%q sale_id=%q: %w", bandID, saleID, err)
	}
	defer rows.Close()

	transactions := make([]applicationmerchbooth.Transaction, 0)
	for rows.Next() {
		var transaction applicationmerchbooth.Transaction
		if err := rows.Scan(
			&transaction.ID,
			&transaction.SaleID,
			&transaction.SaleItemID,
			&transaction.Amount.Amount,
			&transaction.Amount.Currency,
			&transaction.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sale transaction band_id=%q sale_id=%q: %w", bandID, saleID, err)
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sale transactions band_id=%q sale_id=%q: %w", bandID, saleID, err)
	}

	return transactions, nil
}

func scanBoothItem(row pgx.Row) (applicationmerchbooth.BoothItem, error) {
	var variantRow boothVariantRow
	err := row.Scan(
		&variantRow.ProductID,
		&variantRow.VariantID,
		&variantRow.ProductName,
		&variantRow.Category,
		&variantRow.Size,
		&variantRow.Colour,
		&variantRow.PriceAmount,
		&variantRow.CostAmount,
		&variantRow.Currency,
		&variantRow.Quantity,
		&variantRow.PhotoKey,
		&variantRow.PhotoType,
		&variantRow.PhotoSize,
	)
	if err != nil {
		return applicationmerchbooth.BoothItem{}, err
	}

	category, err := inventorydomain.ParseCategory(variantRow.Category)
	if err != nil {
		return applicationmerchbooth.BoothItem{}, err
	}

	size, err := inventorydomain.ParseSize(variantRow.Size)
	if err != nil {
		return applicationmerchbooth.BoothItem{}, err
	}

	return applicationmerchbooth.BoothItem{
		ProductID:   variantRow.ProductID,
		VariantID:   variantRow.VariantID,
		ProductName: variantRow.ProductName,
		Category:    category,
		Size:        size,
		Colour:      variantRow.Colour,
		Price:       inventorydomain.Money{Amount: variantRow.PriceAmount, Currency: variantRow.Currency},
		Cost:        inventorydomain.Money{Amount: variantRow.CostAmount, Currency: variantRow.Currency},
		Quantity:    variantRow.Quantity,
		Photo: inventorydomain.PhotoMetadata{
			ObjectKey:   variantRow.PhotoKey,
			ContentType: variantRow.PhotoType,
			SizeBytes:   variantRow.PhotoSize,
		},
		SoldOut: variantRow.Quantity == 0,
	}, nil
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID string, bandID string, action string, entityType string, entityID string, requestID string, idempotencyKey string, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (id, user_id, band_id, action, entity_type, entity_id, request_id, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, uuid.NewString(), userID, bandID, action, entityType, entityID, requestID, idempotencyKey, createdAt)
	if err != nil {
		return fmt.Errorf("insert merch booth audit log user_id=%q band_id=%q action=%q entity_type=%q entity_id=%q: %w", userID, bandID, action, entityType, entityID, err)
	}

	return nil
}

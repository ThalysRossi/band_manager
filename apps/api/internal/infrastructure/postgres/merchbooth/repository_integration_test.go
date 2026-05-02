package merchbooth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
	postgresinventory "github.com/thalys/band-manager/apps/api/internal/infrastructure/postgres/inventory"
)

func TestRepositoryCreateCashCheckoutFinalizesSale(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Checkout", 4)
	repository := NewRepository(pool)

	sale, err := repository.CreateCashCheckout(ctx, validCashCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2))
	if err != nil {
		t.Fatalf("create cash checkout: %v", err)
	}

	if sale.Status != applicationmerchbooth.SaleStatusFinalized {
		t.Fatalf("expected finalized sale, got %q", sale.Status)
	}

	if sale.Total.Amount != 10000 {
		t.Fatalf("expected total 10000, got %d", sale.Total.Amount)
	}

	if sale.ExpectedProfit.Amount != 6000 {
		t.Fatalf("expected profit 6000, got %d", sale.ExpectedProfit.Amount)
	}

	assertTableCount(t, pool, "sales", "id = $1 AND total_amount = $2", []interface{}{sale.ID, 10000}, 1)
	assertTableCount(t, pool, "sale_items", "sale_id = $1 AND quantity = $2", []interface{}{sale.ID, 2}, 1)
	assertTableCount(t, pool, "payments", "sale_id = $1 AND method = $2 AND status = $3", []interface{}{sale.ID, "cash", "confirmed"}, 1)
	assertTableCount(t, pool, "transactions", "sale_id = $1", []interface{}{sale.ID}, 1)
	assertTableCount(t, pool, "inventory_reservations", "band_id = $1 AND variant_id = $2 AND status = $3", []interface{}{account.BandID, inventoryProduct.Variants[0].ID, "consumed"}, 1)
	assertTableCount(t, pool, "inventory_movements", "band_id = $1 AND variant_id = $2 AND movement_type = $3 AND quantity_delta = $4 AND quantity_after = $5", []interface{}{account.BandID, inventoryProduct.Variants[0].ID, "sale", -2, 2}, 1)
	assertTableCount(t, pool, "merch_variants", "id = $1 AND quantity = $2", []interface{}{inventoryProduct.Variants[0].ID, 2}, 1)
	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2 AND entity_id = $3", []interface{}{account.BandID, "merch_booth.cash_checkout_finalized", sale.ID}, 1)
}

func TestRepositoryCreateCashCheckoutRejectsInsufficientStock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Insufficient", 1)
	repository := NewRepository(pool)

	_, err := repository.CreateCashCheckout(ctx, validCashCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2))
	if !errors.Is(err, applicationmerchbooth.ErrInsufficientStock) {
		t.Fatalf("expected insufficient stock error, got %v", err)
	}

	assertTableCount(t, pool, "sales", "band_id = $1", []interface{}{account.BandID}, 0)
	assertTableCount(t, pool, "merch_variants", "id = $1 AND quantity = $2", []interface{}{inventoryProduct.Variants[0].ID, 1}, 1)
}

func TestRepositoryCreateCashCheckoutRejectsDeletedVariant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Deleted Variant", 2)
	inventoryRepository := postgresinventory.NewRepository(pool)
	err := inventoryRepository.SoftDeleteVariant(ctx, applicationinventory.SoftDeleteVariantCommand{
		Account: applicationinventory.AccountContext{
			UserID: account.UserID,
			BandID: account.BandID,
			Role:   account.Role,
		},
		VariantID:      inventoryProduct.Variants[0].ID,
		IdempotencyKey: "idem_delete_variant",
		RequestID:      "request_delete_variant",
		DeletedAt:      testTimestamp().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("soft delete variant: %v", err)
	}

	repository := NewRepository(pool)
	_, err = repository.CreateCashCheckout(ctx, validCashCheckoutCommand(account, inventoryProduct.Variants[0].ID, 1))
	if !errors.Is(err, applicationmerchbooth.ErrBoothItemNotFound) {
		t.Fatalf("expected booth item not found error, got %v", err)
	}
}

func TestRepositoryCreateCashCheckoutIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Idempotent", 5)
	repository := NewRepository(pool)
	command := validCashCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2)

	firstSale, err := repository.CreateCashCheckout(ctx, command)
	if err != nil {
		t.Fatalf("create first cash checkout: %v", err)
	}

	secondSale, err := repository.CreateCashCheckout(ctx, command)
	if err != nil {
		t.Fatalf("create idempotent cash checkout: %v", err)
	}

	if secondSale.ID != firstSale.ID {
		t.Fatalf("expected idempotent sale id %q, got %q", firstSale.ID, secondSale.ID)
	}

	assertTableCount(t, pool, "sales", "band_id = $1", []interface{}{account.BandID}, 1)
	assertTableCount(t, pool, "merch_variants", "id = $1 AND quantity = $2", []interface{}{inventoryProduct.Variants[0].ID, 3}, 1)

	conflictingCommand := validCashCheckoutCommand(account, inventoryProduct.Variants[0].ID, 1)
	conflictingCommand.IdempotencyKey = command.IdempotencyKey
	_, err = repository.CreateCashCheckout(ctx, conflictingCommand)
	if !errors.Is(err, applicationmerchbooth.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestRepositoryReserveAndCompletePixCheckout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Pix", 4)
	repository := NewRepository(pool)
	command := validPixCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2)

	reservedSale, found, err := repository.ReservePixCheckout(ctx, command)
	if err != nil {
		t.Fatalf("reserve pix checkout: %v", err)
	}
	if found {
		t.Fatal("expected new pix checkout reservation")
	}
	if reservedSale.Status != applicationmerchbooth.SaleStatusPendingPayment {
		t.Fatalf("expected pending sale, got %q", reservedSale.Status)
	}

	assertTableCount(t, pool, "inventory_reservations", "band_id = $1 AND sale_id = $2 AND variant_id = $3 AND status = $4", []interface{}{account.BandID, command.SaleID, inventoryProduct.Variants[0].ID, "reserved"}, 1)
	assertTableCount(t, pool, "merch_variants", "id = $1 AND quantity = $2", []interface{}{inventoryProduct.Variants[0].ID, 4}, 1)
	assertTableCount(t, pool, "transactions", "sale_id = $1", []interface{}{command.SaleID}, 0)

	requestHash, err := applicationmerchbooth.HashPixCheckoutRequest(command)
	if err != nil {
		t.Fatalf("hash pix checkout request: %v", err)
	}

	completedSale, err := repository.CompletePixCheckoutPayment(ctx, applicationmerchbooth.CompletePixCheckoutPaymentCommand{
		Account:     account,
		SaleID:      command.SaleID,
		PaymentID:   command.PaymentID,
		RequestID:   command.RequestID,
		RequestHash: requestHash,
		ProviderResult: applicationmerchbooth.PixPayment{
			Provider:             "mercadopago",
			ProviderOrderID:      "order_1",
			ProviderPaymentID:    "payment_1",
			ProviderReferenceID:  "reference_1",
			ExternalReference:    command.ExternalReference,
			ProviderStatus:       "action_required",
			ProviderStatusDetail: "waiting_transfer",
			LocalStatus:          applicationmerchbooth.PaymentStatusActionRequired,
			Amount:               reservedSale.Total,
			ExpiresAt:            command.ExpiresAt,
			QRCode:               "pix-copy-paste",
			QRCodeBase64:         "base64",
			TicketURL:            "https://example.test/ticket",
			RawProviderResponse:  []byte(`{"id":"order_1"}`),
		},
		IdempotencyKey: command.IdempotencyKey,
		UpdatedAt:      command.CreatedAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("complete pix checkout payment: %v", err)
	}

	if completedSale.Payment.PixQRCode != "pix-copy-paste" {
		t.Fatalf("expected pix qr code, got %q", completedSale.Payment.PixQRCode)
	}
	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2 AND entity_id = $3", []interface{}{account.BandID, "merch_booth.pix_checkout_payment_created", command.PaymentID}, 1)

	idempotentSale, found, err := repository.ReservePixCheckout(ctx, command)
	if err != nil {
		t.Fatalf("load idempotent pix checkout: %v", err)
	}
	if !found {
		t.Fatal("expected idempotent pix checkout")
	}
	if idempotentSale.ID != completedSale.ID {
		t.Fatalf("expected idempotent sale id %q, got %q", completedSale.ID, idempotentSale.ID)
	}
}

func TestRepositoryFailPixCheckoutPaymentCreationReleasesReservation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Pix Fail", 3)
	repository := NewRepository(pool)
	command := validPixCheckoutCommand(account, inventoryProduct.Variants[0].ID, 1)

	_, _, err := repository.ReservePixCheckout(ctx, command)
	if err != nil {
		t.Fatalf("reserve pix checkout: %v", err)
	}

	err = repository.FailPixCheckoutPaymentCreation(ctx, applicationmerchbooth.FailPixCheckoutPaymentCreationCommand{
		Account:        account,
		SaleID:         command.SaleID,
		PaymentID:      command.PaymentID,
		RequestID:      command.RequestID,
		IdempotencyKey: command.IdempotencyKey,
		UpdatedAt:      command.CreatedAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("fail pix checkout payment creation: %v", err)
	}

	assertTableCount(t, pool, "inventory_reservations", "band_id = $1 AND sale_id = $2 AND variant_id = $3 AND status = $4", []interface{}{account.BandID, command.SaleID, inventoryProduct.Variants[0].ID, "released"}, 1)
	assertTableCount(t, pool, "payments", "id = $1 AND status = $2", []interface{}{command.PaymentID, "failed"}, 1)
	assertTableCount(t, pool, "sales", "id = $1 AND status = $2", []interface{}{command.SaleID, "canceled"}, 1)
	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2 AND entity_id = $3", []interface{}{account.BandID, "merch_booth.pix_checkout_payment_creation_failed", command.SaleID}, 1)
}

func TestRepositoryPixReservationReducesAvailableCheckoutStock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	inventoryProduct := createInventoryProduct(t, ctx, pool, account, "Camisa Pix Reserved Stock", 3)
	repository := NewRepository(pool)
	firstCommand := validPixCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2)
	secondCommand := validPixCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2)

	_, _, err := repository.ReservePixCheckout(ctx, firstCommand)
	if err != nil {
		t.Fatalf("reserve first pix checkout: %v", err)
	}

	_, _, err = repository.ReservePixCheckout(ctx, secondCommand)
	if !errors.Is(err, applicationmerchbooth.ErrInsufficientStock) {
		t.Fatalf("expected insufficient stock from active reservation, got %v", err)
	}

	cashCommand := validCashCheckoutCommand(account, inventoryProduct.Variants[0].ID, 2)
	_, err = repository.CreateCashCheckout(ctx, cashCommand)
	if !errors.Is(err, applicationmerchbooth.ErrInsufficientStock) {
		t.Fatalf("expected cash checkout to respect active reservation, got %v", err)
	}
}

func TestRepositoryListBoothItemsIncludesSoldOutVariants(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	createInventoryProduct(t, ctx, pool, account, "Camisa Sold Out", 0)
	repository := NewRepository(pool)

	items, err := repository.ListBoothItems(ctx, applicationmerchbooth.ListBoothItemsQuery{Account: account})
	if err != nil {
		t.Fatalf("list booth items: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected one booth item, got %d", len(items))
	}

	if !items[0].SoldOut {
		t.Fatal("expected sold-out booth item")
	}
}

func newIntegrationDatabase(t *testing.T) (*pgxpool.Pool, applicationmerchbooth.AccountContext) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Skip("DATABASE_URL is not set; skipping Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	setupPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skipf("Postgres is unavailable: %v", err)
	}
	if err := setupPool.Ping(ctx); err != nil {
		setupPool.Close()
		t.Skipf("Postgres is unavailable: %v", err)
	}

	schemaName := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	_, err = setupPool.Exec(ctx, "CREATE SCHEMA "+quotedSchemaName)
	if err != nil {
		setupPool.Close()
		t.Fatalf("create test schema %q: %v", schemaName, err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		setupPool.Close()
		t.Fatalf("parse database url: %v", err)
	}
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName
	poolConfig.MaxConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		setupPool.Close()
		t.Fatalf("create schema-scoped pool: %v", err)
	}

	applyMigrations(ctx, t, pool)
	account := seedAccount(ctx, t, pool)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()

		pool.Close()
		_, dropErr := setupPool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quotedSchemaName+" CASCADE")
		if dropErr != nil {
			t.Logf("drop test schema %q: %v", schemaName, dropErr)
		}
		setupPool.Close()
	})

	return pool, account
}

func applyMigrations(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	for _, migrationPath := range migrationFilePaths(t) {
		body, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", migrationPath, err)
		}

		statements, err := upMigrationStatements(string(body))
		if err != nil {
			t.Fatalf("parse migration %s: %v", migrationPath, err)
		}

		for _, statement := range statements {
			_, err := pool.Exec(ctx, statement)
			if err != nil {
				t.Fatalf("apply migration %s statement %q: %v", migrationPath, statement, err)
			}
		}
	}
}

func migrationFilePaths(t *testing.T) []string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../.."))
	migrationPaths, err := filepath.Glob(filepath.Join(apiRoot, "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migration files: %v", err)
	}
	sort.Strings(migrationPaths)

	return migrationPaths
}

func upMigrationStatements(body string) ([]string, error) {
	upMarkerIndex := strings.Index(body, "-- +goose Up")
	if upMarkerIndex == -1 {
		return nil, errors.New("goose up marker is required")
	}

	downMarkerIndex := strings.Index(body, "-- +goose Down")
	if downMarkerIndex == -1 {
		return nil, errors.New("goose down marker is required")
	}

	if downMarkerIndex <= upMarkerIndex {
		return nil, errors.New("goose down marker must follow up marker")
	}

	upBody := body[upMarkerIndex+len("-- +goose Up") : downMarkerIndex]
	parts := strings.Split(upBody, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}

	return statements, nil
}

func seedAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool) applicationmerchbooth.AccountContext {
	t.Helper()

	now := testTimestamp()
	userID := uuid.NewString()
	bandID := uuid.NewString()

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, auth_provider, auth_provider_user_id, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, userID, "supabase", "auth_"+userID, userID+"@example.com", now)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO bands (id, name, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, bandID, "Os Testes", "America/Recife", now)
	if err != nil {
		t.Fatalf("seed band: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO band_memberships (id, band_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`, uuid.NewString(), bandID, userID, permissions.RoleOwner, now)
	if err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	return applicationmerchbooth.AccountContext{
		UserID: userID,
		BandID: bandID,
		Role:   permissions.RoleOwner,
	}
}

func createInventoryProduct(t *testing.T, ctx context.Context, pool *pgxpool.Pool, account applicationmerchbooth.AccountContext, name string, quantity int) applicationinventory.Product {
	t.Helper()

	repository := postgresinventory.NewRepository(pool)
	product, err := repository.CreateProduct(ctx, applicationinventory.CreateProductCommand{
		Account: applicationinventory.AccountContext{
			UserID: account.UserID,
			BandID: account.BandID,
			Role:   account.Role,
		},
		Name:           name,
		NormalizedName: strings.ToLower(name),
		Category:       inventorydomain.CategoryShirt,
		Photo: inventorydomain.PhotoMetadata{
			ObjectKey:   "bands/test/products/photo.jpg",
			ContentType: "image/jpeg",
			SizeBytes:   1024,
		},
		Variants: []applicationinventory.CreateVariantCommand{
			{
				Size:             inventorydomain.SizeM,
				Colour:           "Preta",
				NormalizedColour: "preta",
				Price:            inventorydomain.Money{Amount: 5000, Currency: "BRL"},
				Cost:             inventorydomain.Money{Amount: 2000, Currency: "BRL"},
				Quantity:         quantity,
			},
		},
		IdempotencyKey: "idem_inventory_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		RequestID:      "request_inventory_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		CreatedAt:      testTimestamp(),
	})
	if err != nil {
		t.Fatalf("create inventory product: %v", err)
	}

	return product
}

func validCashCheckoutCommand(account applicationmerchbooth.AccountContext, variantID string, quantity int) applicationmerchbooth.CreateCashCheckoutCommand {
	return applicationmerchbooth.CreateCashCheckoutCommand{
		Account: account,
		Items: []applicationmerchbooth.CartItem{
			{
				VariantID: variantID,
				Quantity:  quantity,
			},
		},
		IdempotencyKey: "idem_checkout_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		RequestID:      "request_checkout_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		CreatedAt:      testTimestamp().Add(2 * time.Minute),
	}
}

func validPixCheckoutCommand(account applicationmerchbooth.AccountContext, variantID string, quantity int) applicationmerchbooth.CreatePixCheckoutCommand {
	saleID := uuid.NewString()
	return applicationmerchbooth.CreatePixCheckoutCommand{
		Account:           account,
		SaleID:            saleID,
		PaymentID:         uuid.NewString(),
		ExternalReference: "sale_" + saleID,
		Items: []applicationmerchbooth.CartItem{
			{
				VariantID: variantID,
				Quantity:  quantity,
			},
		},
		PayerEmail:     "band@example.com",
		IdempotencyKey: "idem_pix_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		RequestID:      "request_pix_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		CreatedAt:      testTimestamp().Add(3 * time.Minute),
		ExpiresAt:      testTimestamp().Add(33 * time.Minute),
	}
}

func assertTableCount(t *testing.T, pool *pgxpool.Pool, tableName string, whereClause string, args []interface{}, expectedCount int) {
	t.Helper()

	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", pgx.Identifier{tableName}.Sanitize(), whereClause)
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", tableName, err)
	}

	if count != expectedCount {
		t.Fatalf("expected %d rows in %s, got %d", expectedCount, tableName, count)
	}
}

func testTimestamp() time.Time {
	return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
}

package inventory

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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestRepositoryCreateProductWritesVariantMovementAndAuditLog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	product, err := repository.CreateProduct(ctx, validCreateProductCommand(account, "Camisa Logo", 2))
	if err != nil {
		t.Fatalf("create inventory product: %v", err)
	}

	if product.ID == "" {
		t.Fatal("expected created product id")
	}

	if len(product.Variants) != 1 {
		t.Fatalf("expected one variant, got %d", len(product.Variants))
	}

	assertTableCount(t, pool, "merch_products", "band_id = $1", []interface{}{account.BandID}, 1)
	assertTableCount(t, pool, "merch_variants", "band_id = $1", []interface{}{account.BandID}, 1)
	assertTableCount(t, pool, "inventory_movements", "band_id = $1 AND movement_type = $2 AND quantity_delta = $3 AND quantity_after = $4", []interface{}{account.BandID, "initial_stock", 2, 2}, 1)
	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2 AND entity_type = $3", []interface{}{account.BandID, "inventory.product_created", "merch_product"}, 1)
}

func TestRepositoryCreateProductRejectsDuplicateProductIdentity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	_, err := repository.CreateProduct(ctx, validCreateProductCommand(account, "Camisa Logo", 2))
	if err != nil {
		t.Fatalf("create first inventory product: %v", err)
	}

	command := validCreateProductCommand(account, "Camisa Logo", 3)
	command.IdempotencyKey = "idem_duplicate_product"
	_, err = repository.CreateProduct(ctx, command)
	if !errors.Is(err, applicationinventory.ErrDuplicateProduct) {
		t.Fatalf("expected duplicate product error, got %v", err)
	}
}

func TestRepositoryCreateProductRejectsDuplicateVariantIdentity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	command := validCreateProductCommand(account, "Camisa Logo", 2)
	command.Variants = append(command.Variants, command.Variants[0])

	_, err := repository.CreateProduct(ctx, command)
	if !errors.Is(err, applicationinventory.ErrDuplicateVariant) {
		t.Fatalf("expected duplicate variant error, got %v", err)
	}

	assertTableCount(t, pool, "merch_products", "band_id = $1", []interface{}{account.BandID}, 0)
}

func TestRepositoryCreateProductRejectsDatabaseConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		update      func(applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand
		constraints []string
	}{
		{
			name: "negative price",
			update: func(command applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand {
				command.Variants[0].Price.Amount = -1
				return command
			},
			constraints: []string{"merch_variants_price_amount_check"},
		},
		{
			name: "negative cost",
			update: func(command applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand {
				command.Variants[0].Cost.Amount = -1
				return command
			},
			constraints: []string{"merch_variants_cost_amount_check"},
		},
		{
			name: "negative quantity",
			update: func(command applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand {
				command.Variants[0].Quantity = -1
				return command
			},
			constraints: []string{"merch_variants_quantity_check"},
		},
		{
			name: "empty photo object key",
			update: func(command applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand {
				command.Photo.ObjectKey = " "
				return command
			},
			constraints: []string{"merch_products_photo_object_key_present_check"},
		},
		{
			name: "empty photo content type",
			update: func(command applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand {
				command.Photo.ContentType = " "
				return command
			},
			constraints: []string{"merch_products_photo_content_type_present_check"},
		},
		{
			name: "non-positive photo size",
			update: func(command applicationinventory.CreateProductCommand) applicationinventory.CreateProductCommand {
				command.Photo.SizeBytes = 0
				return command
			},
			constraints: []string{"merch_products_photo_size_bytes_check"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			pool, account := newIntegrationDatabase(t)
			repository := NewRepository(pool)

			command := test.update(validCreateProductCommand(account, "Camisa "+uuid.NewString(), 2))
			_, err := repository.CreateProduct(ctx, command)
			if err == nil {
				t.Fatal("expected database constraint error")
			}

			constraintName := postgresConstraintName(err)
			if !containsString(test.constraints, constraintName) {
				t.Fatalf("expected one of constraints %v, got %q from error %v", test.constraints, constraintName, err)
			}
		})
	}
}

func TestRepositoryListInventoryIncludesSoldOutVariants(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	_, err := repository.CreateProduct(ctx, validCreateProductCommand(account, "Camisa Sold Out", 0))
	if err != nil {
		t.Fatalf("create sold-out inventory product: %v", err)
	}

	products, err := repository.ListInventory(ctx, applicationinventory.ListInventoryQuery{Account: account})
	if err != nil {
		t.Fatalf("list inventory: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected one product, got %d", len(products))
	}

	if len(products[0].Variants) != 1 {
		t.Fatalf("expected one variant, got %d", len(products[0].Variants))
	}

	if products[0].Variants[0].Quantity != 0 {
		t.Fatalf("expected sold-out variant quantity 0, got %d", products[0].Variants[0].Quantity)
	}
}

func TestRepositorySoftDeleteProductHidesProductAndVariants(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	product, err := repository.CreateProduct(ctx, validCreateProductCommand(account, "Camisa Deleted", 2))
	if err != nil {
		t.Fatalf("create inventory product: %v", err)
	}

	err = repository.SoftDeleteProduct(ctx, applicationinventory.SoftDeleteProductCommand{
		Account:        account,
		ProductID:      product.ID,
		IdempotencyKey: "idem_delete_product",
		RequestID:      "request_delete_product",
		DeletedAt:      testTimestamp().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("soft delete product: %v", err)
	}

	products, err := repository.ListInventory(ctx, applicationinventory.ListInventoryQuery{Account: account})
	if err != nil {
		t.Fatalf("list inventory after product delete: %v", err)
	}

	if len(products) != 0 {
		t.Fatalf("expected deleted product to be hidden, got %d products", len(products))
	}

	assertTableCount(t, pool, "merch_variants", "band_id = $1 AND deleted_at IS NOT NULL", []interface{}{account.BandID}, 1)
	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2", []interface{}{account.BandID, "inventory.product_deleted"}, 1)
}

func TestRepositorySoftDeleteVariantHidesVariant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	command := validCreateProductCommand(account, "Camisa Variant Deleted", 2)
	command.Variants = append(command.Variants, applicationinventory.CreateVariantCommand{
		Size:             inventorydomain.SizeG,
		Colour:           "Preta",
		NormalizedColour: "preta",
		Price:            inventorydomain.Money{Amount: 5000, Currency: "BRL"},
		Cost:             inventorydomain.Money{Amount: 2000, Currency: "BRL"},
		Quantity:         3,
	})

	product, err := repository.CreateProduct(ctx, command)
	if err != nil {
		t.Fatalf("create inventory product: %v", err)
	}

	err = repository.SoftDeleteVariant(ctx, applicationinventory.SoftDeleteVariantCommand{
		Account:        account,
		VariantID:      product.Variants[0].ID,
		IdempotencyKey: "idem_delete_variant",
		RequestID:      "request_delete_variant",
		DeletedAt:      testTimestamp().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("soft delete variant: %v", err)
	}

	products, err := repository.ListInventory(ctx, applicationinventory.ListInventoryQuery{Account: account})
	if err != nil {
		t.Fatalf("list inventory after variant delete: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected product to remain visible, got %d products", len(products))
	}

	if len(products[0].Variants) != 1 {
		t.Fatalf("expected one remaining variant, got %d", len(products[0].Variants))
	}

	if products[0].Variants[0].ID == product.Variants[0].ID {
		t.Fatal("expected deleted variant to be hidden")
	}

	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2", []interface{}{account.BandID, "inventory.variant_deleted"}, 1)
}

func TestRepositoryUpdateProductWritesAuditLog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	product, err := repository.CreateProduct(ctx, validCreateProductCommand(account, "Camisa Update Product", 2))
	if err != nil {
		t.Fatalf("create inventory product: %v", err)
	}

	updatedProduct, err := repository.UpdateProduct(ctx, applicationinventory.UpdateProductCommand{
		Account:        account,
		ProductID:      product.ID,
		Name:           "Camisa Atualizada",
		NormalizedName: "camisa atualizada",
		Category:       inventorydomain.CategoryShirt,
		Photo: inventorydomain.PhotoMetadata{
			ObjectKey:   "bands/test/products/updated.jpg",
			ContentType: "image/jpeg",
			SizeBytes:   2048,
		},
		IdempotencyKey: "idem_update_product",
		RequestID:      "request_update_product",
		UpdatedAt:      testTimestamp().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update inventory product: %v", err)
	}

	if updatedProduct.Name != "Camisa Atualizada" {
		t.Fatalf("expected updated product name, got %q", updatedProduct.Name)
	}

	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2", []interface{}{account.BandID, "inventory.product_updated"}, 1)
}

func TestRepositoryUpdateVariantWritesManualAdjustmentMovementAndAuditLog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	product, err := repository.CreateProduct(ctx, validCreateProductCommand(account, "Camisa Update Variant", 2))
	if err != nil {
		t.Fatalf("create inventory product: %v", err)
	}

	variantID := product.Variants[0].ID
	updatedVariant, err := repository.UpdateVariant(ctx, applicationinventory.UpdateVariantCommand{
		Account:          account,
		VariantID:        variantID,
		Size:             inventorydomain.SizeM,
		Colour:           "Preta",
		NormalizedColour: "preta",
		Price:            inventorydomain.Money{Amount: 5500, Currency: "BRL"},
		Cost:             inventorydomain.Money{Amount: 2500, Currency: "BRL"},
		Quantity:         5,
		IdempotencyKey:   "idem_update_variant",
		RequestID:        "request_update_variant",
		UpdatedAt:        testTimestamp().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update inventory variant: %v", err)
	}

	if updatedVariant.Quantity != 5 {
		t.Fatalf("expected updated quantity 5, got %d", updatedVariant.Quantity)
	}

	assertTableCount(t, pool, "inventory_movements", "band_id = $1 AND variant_id = $2 AND movement_type = $3 AND quantity_delta = $4 AND quantity_after = $5", []interface{}{account.BandID, variantID, "manual_adjustment", 3, 5}, 1)
	assertTableCount(t, pool, "audit_logs", "band_id = $1 AND action = $2 AND entity_id = $3", []interface{}{account.BandID, "inventory.variant_updated", variantID}, 1)
}

func newIntegrationDatabase(t *testing.T) (*pgxpool.Pool, applicationinventory.AccountContext) {
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

	migrationPaths := migrationFilePaths(t)
	for _, migrationPath := range migrationPaths {
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
	if len(migrationPaths) == 0 {
		t.Fatal("expected migration files")
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

func seedAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool) applicationinventory.AccountContext {
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

	return applicationinventory.AccountContext{
		UserID: userID,
		BandID: bandID,
		Role:   permissions.RoleOwner,
	}
}

func validCreateProductCommand(account applicationinventory.AccountContext, name string, quantity int) applicationinventory.CreateProductCommand {
	return applicationinventory.CreateProductCommand{
		Account:        account,
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
		IdempotencyKey: "idem_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		RequestID:      "request_" + strings.ReplaceAll(uuid.NewString(), "-", "_"),
		CreatedAt:      testTimestamp(),
	}
}

func assertTableCount(t *testing.T, pool *pgxpool.Pool, tableName string, whereClause string, args []interface{}, expectedCount int) {
	t.Helper()

	ctx := context.Background()
	var count int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", pgx.Identifier{tableName}.Sanitize(), whereClause)
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", tableName, err)
	}

	if count != expectedCount {
		t.Fatalf("expected %d rows in %s, got %d", expectedCount, tableName, count)
	}
}

func postgresConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}

	for unwrappedErr := errors.Unwrap(err); unwrappedErr != nil; unwrappedErr = errors.Unwrap(unwrappedErr) {
		if errors.As(unwrappedErr, &pgErr) {
			return pgErr.ConstraintName
		}
	}

	return ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func testTimestamp() time.Time {
	return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
}

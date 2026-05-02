package financialreports

import (
	"context"
	"errors"
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
	applicationfinancialreports "github.com/thalys/band-manager/apps/api/internal/application/financialreports"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestRepositoryGetReportAggregatesFinalizedConfirmedSales(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	shirtProductID, shirtVariantID := seedProduct(t, ctx, pool, account.BandID, "Camisa Historica", "shirt")
	vinylProductID, vinylVariantID := seedProduct(t, ctx, pool, account.BandID, "Vinil Historico", "vinyl")
	seedSale(t, ctx, pool, saleSeed{
		Account:       account,
		ProductID:     shirtProductID,
		VariantID:     shirtVariantID,
		ProductName:   "Camisa vendida",
		Category:      "shirt",
		Quantity:      2,
		UnitPrice:     5000,
		UnitCost:      2000,
		PaymentMethod: "cash",
		PaymentStatus: "confirmed",
		SaleStatus:    "finalized",
		FinalizedAt:   time.Date(2026, 5, 1, 15, 0, 0, 0, time.UTC),
	})
	seedSale(t, ctx, pool, saleSeed{
		Account:       account,
		ProductID:     shirtProductID,
		VariantID:     shirtVariantID,
		ProductName:   "Camisa vendida",
		Category:      "shirt",
		Quantity:      1,
		UnitPrice:     3000,
		UnitCost:      1000,
		PaymentMethod: "pix",
		PaymentStatus: "confirmed",
		SaleStatus:    "finalized",
		FinalizedAt:   time.Date(2026, 5, 2, 15, 30, 0, 0, time.UTC),
	})
	seedSale(t, ctx, pool, saleSeed{
		Account:       account,
		ProductID:     vinylProductID,
		VariantID:     vinylVariantID,
		ProductName:   "Vinil vendido",
		Category:      "vinyl",
		Quantity:      1,
		UnitPrice:     7000,
		UnitCost:      5000,
		PaymentMethod: "card",
		PaymentStatus: "confirmed",
		SaleStatus:    "finalized",
		FinalizedAt:   time.Date(2026, 5, 3, 14, 0, 0, 0, time.UTC),
	})

	report, err := repository.GetReport(ctx, reportQuery(account, "2026-05-01", "2026-05-03"))
	if err != nil {
		t.Fatalf("get report: %v", err)
	}

	if report.Summary.SaleCount != 3 {
		t.Fatalf("expected 3 sales, got %d", report.Summary.SaleCount)
	}
	if report.Summary.ItemCount != 4 {
		t.Fatalf("expected 4 items, got %d", report.Summary.ItemCount)
	}
	if report.Summary.GrossRevenue.Amount != 20000 {
		t.Fatalf("expected gross 20000, got %d", report.Summary.GrossRevenue.Amount)
	}
	if report.Summary.TotalHistoricalCost.Amount != 10000 {
		t.Fatalf("expected cost 10000, got %d", report.Summary.TotalHistoricalCost.Amount)
	}
	if report.Summary.ExpectedProfit.Amount != 10000 {
		t.Fatalf("expected profit 10000, got %d", report.Summary.ExpectedProfit.Amount)
	}
	if len(report.PaymentMethods) != 3 {
		t.Fatalf("expected three payment methods, got %d", len(report.PaymentMethods))
	}
	if len(report.Categories) != 2 {
		t.Fatalf("expected two categories, got %d", len(report.Categories))
	}
	if len(report.Products) != 2 {
		t.Fatalf("expected two products, got %d", len(report.Products))
	}
	if len(report.Days) != 3 {
		t.Fatalf("expected three day buckets, got %d", len(report.Days))
	}
	if report.Days[1].Date != "2026-05-02" {
		t.Fatalf("expected second bucket to use local finalized date 2026-05-02, got %q", report.Days[1].Date)
	}
}

func TestRepositoryGetReportExcludesNonFinalizedAndUnconfirmedPayments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pool, account := newIntegrationDatabase(t)
	repository := NewRepository(pool)

	productID, variantID := seedProduct(t, ctx, pool, account.BandID, "Camisa Filtros", "shirt")
	validSeed := saleSeed{
		Account:       account,
		ProductID:     productID,
		VariantID:     variantID,
		ProductName:   "Camisa incluida",
		Category:      "shirt",
		Quantity:      1,
		UnitPrice:     4000,
		UnitCost:      1500,
		PaymentMethod: "cash",
		PaymentStatus: "confirmed",
		SaleStatus:    "finalized",
		FinalizedAt:   time.Date(2026, 5, 1, 15, 0, 0, 0, time.UTC),
	}
	seedSale(t, ctx, pool, validSeed)

	unconfirmedSeed := validSeed
	unconfirmedSeed.ProductName = "Pix pendente"
	unconfirmedSeed.PaymentMethod = "pix"
	unconfirmedSeed.PaymentStatus = "action_required"
	seedSale(t, ctx, pool, unconfirmedSeed)

	canceledPaymentSeed := validSeed
	canceledPaymentSeed.ProductName = "Pix cancelado"
	canceledPaymentSeed.PaymentMethod = "pix"
	canceledPaymentSeed.PaymentStatus = "canceled"
	seedSale(t, ctx, pool, canceledPaymentSeed)

	pendingSaleSeed := validSeed
	pendingSaleSeed.ProductName = "Venda pendente"
	pendingSaleSeed.SaleStatus = "pending_payment"
	pendingSaleSeed.FinalizedAt = time.Time{}
	seedSale(t, ctx, pool, pendingSaleSeed)

	canceledSaleSeed := validSeed
	canceledSaleSeed.ProductName = "Venda cancelada"
	canceledSaleSeed.SaleStatus = "canceled"
	seedSale(t, ctx, pool, canceledSaleSeed)

	outsideRangeSeed := validSeed
	outsideRangeSeed.ProductName = "Venda antiga"
	outsideRangeSeed.FinalizedAt = time.Date(2026, 4, 30, 2, 59, 59, 0, time.UTC)
	seedSale(t, ctx, pool, outsideRangeSeed)

	report, err := repository.GetReport(ctx, reportQuery(account, "2026-05-01", "2026-05-01"))
	if err != nil {
		t.Fatalf("get report: %v", err)
	}

	if report.Summary.SaleCount != 1 {
		t.Fatalf("expected one included sale, got %d", report.Summary.SaleCount)
	}
	if report.Summary.GrossRevenue.Amount != 4000 {
		t.Fatalf("expected gross 4000, got %d", report.Summary.GrossRevenue.Amount)
	}
	if len(report.PaymentMethods) != 1 || report.PaymentMethods[0].Method != applicationfinancialreports.PaymentMethodCash {
		t.Fatalf("expected only cash payment method, got %#v", report.PaymentMethods)
	}
}

type saleSeed struct {
	Account       applicationfinancialreports.AccountContext
	ProductID     string
	VariantID     string
	ProductName   string
	Category      string
	Quantity      int
	UnitPrice     int
	UnitCost      int
	PaymentMethod string
	PaymentStatus string
	SaleStatus    string
	FinalizedAt   time.Time
}

func newIntegrationDatabase(t *testing.T) (*pgxpool.Pool, applicationfinancialreports.AccountContext) {
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

func seedAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool) applicationfinancialreports.AccountContext {
	t.Helper()

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
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

	return applicationfinancialreports.AccountContext{
		UserID:       userID,
		BandID:       bandID,
		BandTimezone: "America/Recife",
		Role:         permissions.RoleOwner,
	}
}

func seedProduct(t *testing.T, ctx context.Context, pool *pgxpool.Pool, bandID string, name string, category string) (string, string) {
	t.Helper()

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	productID := uuid.NewString()
	variantID := uuid.NewString()
	_, err := pool.Exec(ctx, `
		INSERT INTO merch_products (
			id, band_id, name, normalized_name, category, photo_object_key,
			photo_content_type, photo_size_bytes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, productID, bandID, name, strings.ToLower(name), category, "bands/test/products/photo.jpg", "image/jpeg", 1024, now)
	if err != nil {
		t.Fatalf("seed product: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO merch_variants (
			id, band_id, product_id, size, colour, normalized_colour,
			price_amount, cost_amount, currency, quantity, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
	`, variantID, bandID, productID, "m", "Preta", "preta", 5000, 2000, "BRL", 10, now)
	if err != nil {
		t.Fatalf("seed variant: %v", err)
	}

	return productID, variantID
}

func seedSale(t *testing.T, ctx context.Context, pool *pgxpool.Pool, seed saleSeed) {
	t.Helper()

	createdAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	saleID := uuid.NewString()
	saleItemID := uuid.NewString()
	paymentID := uuid.NewString()
	lineTotal := seed.UnitPrice * seed.Quantity
	lineCost := seed.UnitCost * seed.Quantity
	expectedProfit := lineTotal - lineCost

	_, err := pool.Exec(ctx, `
		INSERT INTO sales (
			id, band_id, created_by_user_id, status, total_amount,
			expected_profit_amount, currency, finalized_at, idempotency_key,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	`, saleID, seed.Account.BandID, seed.Account.UserID, seed.SaleStatus, lineTotal, expectedProfit, "BRL", nullableTime(seed.FinalizedAt), "idem_"+saleID, createdAt)
	if err != nil {
		t.Fatalf("seed sale: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO sale_items (
			id, sale_id, band_id, product_id, variant_id, product_name,
			category, size, colour, quantity, unit_price_amount, unit_cost_amount,
			line_total_amount, expected_profit_amount, currency, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, saleItemID, saleID, seed.Account.BandID, seed.ProductID, seed.VariantID, seed.ProductName, seed.Category, "m", "Preta", seed.Quantity, seed.UnitPrice, seed.UnitCost, lineTotal, expectedProfit, "BRL", createdAt)
	if err != nil {
		t.Fatalf("seed sale item: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO payments (
			id, sale_id, band_id, method, status, amount_minor,
			currency, confirmed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, paymentID, saleID, seed.Account.BandID, seed.PaymentMethod, seed.PaymentStatus, lineTotal, "BRL", confirmedAt(seed.PaymentStatus, seed.FinalizedAt), createdAt)
	if err != nil {
		t.Fatalf("seed payment: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO transactions (
			id, sale_id, sale_item_id, band_id, transaction_type,
			amount_minor, currency, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.NewString(), saleID, saleItemID, seed.Account.BandID, "sale_item", lineTotal, "BRL", createdAt)
	if err != nil {
		t.Fatalf("seed transaction: %v", err)
	}
}

func reportQuery(account applicationfinancialreports.AccountContext, from string, to string) applicationfinancialreports.ReportQuery {
	location := time.FixedZone("America/Recife", -3*60*60)
	fromLocal, _ := time.ParseInLocation("2006-01-02", from, location)
	toLocal, _ := time.ParseInLocation("2006-01-02", to, location)
	return applicationfinancialreports.ReportQuery{
		Account: account,
		Range: applicationfinancialreports.ReportRange{
			From:     from,
			To:       to,
			Timezone: "America/Recife",
		},
		FromUTC:        fromLocal.UTC(),
		ToExclusiveUTC: toLocal.AddDate(0, 0, 1).UTC(),
	}
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}

	return value
}

func confirmedAt(paymentStatus string, finalizedAt time.Time) interface{} {
	if paymentStatus != "confirmed" {
		return nil
	}

	if finalizedAt.IsZero() {
		return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	}

	return finalizedAt
}

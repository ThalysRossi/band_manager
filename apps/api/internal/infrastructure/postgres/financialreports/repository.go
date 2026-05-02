package financialreports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	applicationfinancialreports "github.com/thalys/band-manager/apps/api/internal/application/financialreports"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
)

const reportLinesCTE = `
WITH confirmed_payments AS (
	SELECT DISTINCT ON (sale_id)
		sale_id,
		method
	FROM payments
	WHERE band_id = $1
		AND status = 'confirmed'
		AND confirmed_at IS NOT NULL
		AND method IN ('cash', 'pix', 'card')
	ORDER BY sale_id, confirmed_at DESC, id DESC
),
report_lines AS (
	SELECT
		sales.id AS sale_id,
		confirmed_payments.method AS payment_method,
		(sales.finalized_at AT TIME ZONE $4)::date::text AS finalized_local_date,
		sale_items.product_id::text AS product_id,
		sale_items.product_name,
		sale_items.category,
		sale_items.quantity,
		transactions.amount_minor AS line_total_amount,
		sale_items.unit_cost_amount * sale_items.quantity AS total_cost_amount,
		transactions.amount_minor - (sale_items.unit_cost_amount * sale_items.quantity) AS expected_profit_amount
	FROM sales
	INNER JOIN confirmed_payments ON confirmed_payments.sale_id = sales.id
	INNER JOIN sale_items ON sale_items.sale_id = sales.id
	INNER JOIN transactions ON transactions.sale_item_id = sale_items.id
	WHERE sales.band_id = $1
		AND sales.status = 'finalized'
		AND sales.finalized_at IS NOT NULL
		AND sales.finalized_at >= $2
		AND sales.finalized_at < $3
		AND sale_items.band_id = $1
		AND transactions.band_id = $1
		AND transactions.transaction_type = 'sale_item'
)
`

type Repository struct {
	pool *pgxpool.Pool
}

type aggregateRow struct {
	SaleCount           int64
	ItemCount           int64
	GrossRevenue        int64
	TotalHistoricalCost int64
	ExpectedProfit      int64
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (repository Repository) GetReport(ctx context.Context, query applicationfinancialreports.ReportQuery) (applicationfinancialreports.Report, error) {
	summary, err := repository.getSummary(ctx, query)
	if err != nil {
		return applicationfinancialreports.Report{}, err
	}

	paymentMethods, err := repository.getPaymentMethods(ctx, query)
	if err != nil {
		return applicationfinancialreports.Report{}, err
	}

	categories, err := repository.getCategories(ctx, query)
	if err != nil {
		return applicationfinancialreports.Report{}, err
	}

	products, err := repository.getProducts(ctx, query)
	if err != nil {
		return applicationfinancialreports.Report{}, err
	}

	days, err := repository.getDays(ctx, query)
	if err != nil {
		return applicationfinancialreports.Report{}, err
	}

	return applicationfinancialreports.Report{
		Range:          query.Range,
		Summary:        toReportSummary(summary),
		PaymentMethods: paymentMethods,
		Categories:     categories,
		Products:       products,
		Days:           days,
	}, nil
}

func (repository Repository) getSummary(ctx context.Context, query applicationfinancialreports.ReportQuery) (aggregateRow, error) {
	var row aggregateRow
	err := repository.pool.QueryRow(ctx, reportLinesCTE+`
		SELECT
			count(DISTINCT sale_id),
			COALESCE(sum(quantity), 0),
			COALESCE(sum(line_total_amount), 0),
			COALESCE(sum(total_cost_amount), 0),
			COALESCE(sum(expected_profit_amount), 0)
		FROM report_lines
	`, query.Account.BandID, query.FromUTC, query.ToExclusiveUTC, query.Range.Timezone).Scan(&row.SaleCount, &row.ItemCount, &row.GrossRevenue, &row.TotalHistoricalCost, &row.ExpectedProfit)
	if err != nil {
		return aggregateRow{}, fmt.Errorf("query financial report summary band_id=%q from_utc=%s to_exclusive_utc=%s: %w", query.Account.BandID, formatUTC(query.FromUTC), formatUTC(query.ToExclusiveUTC), err)
	}

	return row, nil
}

func (repository Repository) getPaymentMethods(ctx context.Context, query applicationfinancialreports.ReportQuery) ([]applicationfinancialreports.PaymentMethodBreakdown, error) {
	rows, err := repository.pool.Query(ctx, reportLinesCTE+`
		SELECT
			payment_method,
			count(DISTINCT sale_id),
			COALESCE(sum(quantity), 0),
			COALESCE(sum(line_total_amount), 0),
			COALESCE(sum(total_cost_amount), 0),
			COALESCE(sum(expected_profit_amount), 0)
		FROM report_lines
		GROUP BY payment_method
		ORDER BY payment_method ASC
	`, query.Account.BandID, query.FromUTC, query.ToExclusiveUTC, query.Range.Timezone)
	if err != nil {
		return nil, fmt.Errorf("query financial report payment methods band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	breakdowns := make([]applicationfinancialreports.PaymentMethodBreakdown, 0)
	for rows.Next() {
		var method string
		var row aggregateRow
		if err := rows.Scan(&method, &row.SaleCount, &row.ItemCount, &row.GrossRevenue, &row.TotalHistoricalCost, &row.ExpectedProfit); err != nil {
			return nil, fmt.Errorf("scan financial report payment method band_id=%q: %w", query.Account.BandID, err)
		}

		breakdowns = append(breakdowns, applicationfinancialreports.PaymentMethodBreakdown{
			Method:              applicationfinancialreports.PaymentMethod(method),
			SaleCount:           int(row.SaleCount),
			ItemCount:           int(row.ItemCount),
			GrossRevenue:        money(row.GrossRevenue),
			TotalHistoricalCost: money(row.TotalHistoricalCost),
			ExpectedProfit:      money(row.ExpectedProfit),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate financial report payment methods band_id=%q: %w", query.Account.BandID, err)
	}

	return breakdowns, nil
}

func (repository Repository) getCategories(ctx context.Context, query applicationfinancialreports.ReportQuery) ([]applicationfinancialreports.CategoryBreakdown, error) {
	rows, err := repository.pool.Query(ctx, reportLinesCTE+`
		SELECT
			category,
			count(DISTINCT sale_id),
			COALESCE(sum(quantity), 0),
			COALESCE(sum(line_total_amount), 0),
			COALESCE(sum(total_cost_amount), 0),
			COALESCE(sum(expected_profit_amount), 0)
		FROM report_lines
		GROUP BY category
		ORDER BY category ASC
	`, query.Account.BandID, query.FromUTC, query.ToExclusiveUTC, query.Range.Timezone)
	if err != nil {
		return nil, fmt.Errorf("query financial report categories band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	breakdowns := make([]applicationfinancialreports.CategoryBreakdown, 0)
	for rows.Next() {
		var category string
		var row aggregateRow
		if err := rows.Scan(&category, &row.SaleCount, &row.ItemCount, &row.GrossRevenue, &row.TotalHistoricalCost, &row.ExpectedProfit); err != nil {
			return nil, fmt.Errorf("scan financial report category band_id=%q: %w", query.Account.BandID, err)
		}

		parsedCategory, err := inventorydomain.ParseCategory(category)
		if err != nil {
			return nil, fmt.Errorf("parse financial report category band_id=%q category=%q: %w", query.Account.BandID, category, err)
		}

		breakdowns = append(breakdowns, applicationfinancialreports.CategoryBreakdown{
			Category:            parsedCategory,
			SaleCount:           int(row.SaleCount),
			ItemCount:           int(row.ItemCount),
			GrossRevenue:        money(row.GrossRevenue),
			TotalHistoricalCost: money(row.TotalHistoricalCost),
			ExpectedProfit:      money(row.ExpectedProfit),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate financial report categories band_id=%q: %w", query.Account.BandID, err)
	}

	return breakdowns, nil
}

func (repository Repository) getProducts(ctx context.Context, query applicationfinancialreports.ReportQuery) ([]applicationfinancialreports.ProductBreakdown, error) {
	rows, err := repository.pool.Query(ctx, reportLinesCTE+`
		SELECT
			product_id,
			product_name,
			category,
			count(DISTINCT sale_id),
			COALESCE(sum(quantity), 0),
			COALESCE(sum(line_total_amount), 0),
			COALESCE(sum(total_cost_amount), 0),
			COALESCE(sum(expected_profit_amount), 0)
		FROM report_lines
		GROUP BY product_id, product_name, category
		ORDER BY product_name ASC, product_id ASC
	`, query.Account.BandID, query.FromUTC, query.ToExclusiveUTC, query.Range.Timezone)
	if err != nil {
		return nil, fmt.Errorf("query financial report products band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	breakdowns := make([]applicationfinancialreports.ProductBreakdown, 0)
	for rows.Next() {
		var productID string
		var productName string
		var category string
		var row aggregateRow
		if err := rows.Scan(&productID, &productName, &category, &row.SaleCount, &row.ItemCount, &row.GrossRevenue, &row.TotalHistoricalCost, &row.ExpectedProfit); err != nil {
			return nil, fmt.Errorf("scan financial report product band_id=%q: %w", query.Account.BandID, err)
		}

		parsedCategory, err := inventorydomain.ParseCategory(category)
		if err != nil {
			return nil, fmt.Errorf("parse financial report product category band_id=%q product_id=%q category=%q: %w", query.Account.BandID, productID, category, err)
		}

		breakdowns = append(breakdowns, applicationfinancialreports.ProductBreakdown{
			ProductID:           productID,
			ProductName:         productName,
			Category:            parsedCategory,
			SaleCount:           int(row.SaleCount),
			ItemCount:           int(row.ItemCount),
			GrossRevenue:        money(row.GrossRevenue),
			TotalHistoricalCost: money(row.TotalHistoricalCost),
			ExpectedProfit:      money(row.ExpectedProfit),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate financial report products band_id=%q: %w", query.Account.BandID, err)
	}

	return breakdowns, nil
}

func (repository Repository) getDays(ctx context.Context, query applicationfinancialreports.ReportQuery) ([]applicationfinancialreports.DayBreakdown, error) {
	rows, err := repository.pool.Query(ctx, reportLinesCTE+`
		SELECT
			finalized_local_date,
			count(DISTINCT sale_id),
			COALESCE(sum(quantity), 0),
			COALESCE(sum(line_total_amount), 0),
			COALESCE(sum(total_cost_amount), 0),
			COALESCE(sum(expected_profit_amount), 0)
		FROM report_lines
		GROUP BY finalized_local_date
		ORDER BY finalized_local_date ASC
	`, query.Account.BandID, query.FromUTC, query.ToExclusiveUTC, query.Range.Timezone)
	if err != nil {
		return nil, fmt.Errorf("query financial report days band_id=%q: %w", query.Account.BandID, err)
	}
	defer rows.Close()

	breakdowns := make([]applicationfinancialreports.DayBreakdown, 0)
	for rows.Next() {
		var date string
		var row aggregateRow
		if err := rows.Scan(&date, &row.SaleCount, &row.ItemCount, &row.GrossRevenue, &row.TotalHistoricalCost, &row.ExpectedProfit); err != nil {
			return nil, fmt.Errorf("scan financial report day band_id=%q: %w", query.Account.BandID, err)
		}

		breakdowns = append(breakdowns, applicationfinancialreports.DayBreakdown{
			Date:                date,
			SaleCount:           int(row.SaleCount),
			ItemCount:           int(row.ItemCount),
			GrossRevenue:        money(row.GrossRevenue),
			TotalHistoricalCost: money(row.TotalHistoricalCost),
			ExpectedProfit:      money(row.ExpectedProfit),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate financial report days band_id=%q: %w", query.Account.BandID, err)
	}

	return breakdowns, nil
}

func toReportSummary(row aggregateRow) applicationfinancialreports.ReportSummary {
	return applicationfinancialreports.ReportSummary{
		SaleCount:           int(row.SaleCount),
		ItemCount:           int(row.ItemCount),
		GrossRevenue:        money(row.GrossRevenue),
		TotalHistoricalCost: money(row.TotalHistoricalCost),
		ExpectedProfit:      money(row.ExpectedProfit),
	}
}

func money(amount int64) inventorydomain.Money {
	return inventorydomain.Money{Amount: int(amount), Currency: "BRL"}
}

func formatUTC(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

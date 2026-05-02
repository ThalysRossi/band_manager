package financialreports

import (
	"context"
	"fmt"
	"strings"
	"time"

	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

const reportDateLayout = "2006-01-02"

type Repository interface {
	GetReport(ctx context.Context, query ReportQuery) (Report, error)
}

type AccountContext struct {
	UserID       string
	BandID       string
	BandTimezone string
	Role         permissions.Role
}

type ListReportInput struct {
	Account AccountContext
	From    string
	To      string
	Now     time.Time
}

type ReportQuery struct {
	Account        AccountContext
	Range          ReportRange
	FromUTC        time.Time
	ToExclusiveUTC time.Time
}

type ReportRange struct {
	From     string
	To       string
	Timezone string
}

type Report struct {
	Range          ReportRange
	Summary        ReportSummary
	PaymentMethods []PaymentMethodBreakdown
	Categories     []CategoryBreakdown
	Products       []ProductBreakdown
	Days           []DayBreakdown
}

type ReportSummary struct {
	SaleCount           int
	ItemCount           int
	GrossRevenue        inventorydomain.Money
	TotalHistoricalCost inventorydomain.Money
	ExpectedProfit      inventorydomain.Money
}

type PaymentMethod string

const (
	PaymentMethodCash PaymentMethod = "cash"
	PaymentMethodPix  PaymentMethod = "pix"
	PaymentMethodCard PaymentMethod = "card"
)

type PaymentMethodBreakdown struct {
	Method              PaymentMethod
	SaleCount           int
	ItemCount           int
	GrossRevenue        inventorydomain.Money
	TotalHistoricalCost inventorydomain.Money
	ExpectedProfit      inventorydomain.Money
}

type CategoryBreakdown struct {
	Category            inventorydomain.Category
	SaleCount           int
	ItemCount           int
	GrossRevenue        inventorydomain.Money
	TotalHistoricalCost inventorydomain.Money
	ExpectedProfit      inventorydomain.Money
}

type ProductBreakdown struct {
	ProductID           string
	ProductName         string
	Category            inventorydomain.Category
	SaleCount           int
	ItemCount           int
	GrossRevenue        inventorydomain.Money
	TotalHistoricalCost inventorydomain.Money
	ExpectedProfit      inventorydomain.Money
}

type DayBreakdown struct {
	Date                string
	SaleCount           int
	ItemCount           int
	GrossRevenue        inventorydomain.Money
	TotalHistoricalCost inventorydomain.Money
	ExpectedProfit      inventorydomain.Money
}

func ListReport(ctx context.Context, repository Repository, input ListReportInput) (Report, error) {
	query, err := validateListReportInput(input)
	if err != nil {
		return Report{}, err
	}

	report, err := repository.GetReport(ctx, query)
	if err != nil {
		return Report{}, fmt.Errorf("get financial report band_id=%q from=%q to=%q timezone=%q: %w", query.Account.BandID, query.Range.From, query.Range.To, query.Range.Timezone, err)
	}

	return report, nil
}

func validateListReportInput(input ListReportInput) (ReportQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return ReportQuery{}, err
	}

	if input.Now.IsZero() {
		return ReportQuery{}, fmt.Errorf("now timestamp is required")
	}

	location, err := time.LoadLocation(input.Account.BandTimezone)
	if err != nil {
		return ReportQuery{}, fmt.Errorf("band timezone %q is invalid: %w", input.Account.BandTimezone, err)
	}

	localNow := input.Now.In(location)
	localTo := dateOnly(localNow)
	if strings.TrimSpace(input.To) != "" {
		localTo, err = parseLocalDate("to", input.To, location)
		if err != nil {
			return ReportQuery{}, err
		}
	}

	localFrom := localTo.AddDate(0, -3, 0)
	if strings.TrimSpace(input.From) != "" {
		localFrom, err = parseLocalDate("from", input.From, location)
		if err != nil {
			return ReportQuery{}, err
		}
	}

	if localFrom.After(localTo) {
		return ReportQuery{}, fmt.Errorf("from date %q must be on or before to date %q", localFrom.Format(reportDateLayout), localTo.Format(reportDateLayout))
	}

	return ReportQuery{
		Account: input.Account,
		Range: ReportRange{
			From:     localFrom.Format(reportDateLayout),
			To:       localTo.Format(reportDateLayout),
			Timezone: input.Account.BandTimezone,
		},
		FromUTC:        localFrom.UTC(),
		ToExclusiveUTC: localTo.AddDate(0, 0, 1).UTC(),
	}, nil
}

func parseLocalDate(label string, value string, location *time.Location) (time.Time, error) {
	trimmedValue := strings.TrimSpace(value)
	parsed, err := time.ParseInLocation(reportDateLayout, trimmedValue, location)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s date %q must use YYYY-MM-DD", label, value)
	}

	if parsed.Format(reportDateLayout) != trimmedValue {
		return time.Time{}, fmt.Errorf("%s date %q must use YYYY-MM-DD", label, value)
	}

	return parsed, nil
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func validateReadAccount(account AccountContext) error {
	if strings.TrimSpace(account.UserID) == "" {
		return fmt.Errorf("user id is required")
	}

	if strings.TrimSpace(account.BandID) == "" {
		return fmt.Errorf("band id is required")
	}

	if strings.TrimSpace(account.BandTimezone) == "" {
		return fmt.Errorf("band timezone is required")
	}

	if !permissions.CanReadInAlpha(account.Role) {
		return fmt.Errorf("alpha read access denied for role %q", account.Role)
	}

	return nil
}

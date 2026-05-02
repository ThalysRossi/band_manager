package financialreportshandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationfinancialreports "github.com/thalys/band-manager/apps/api/internal/application/financialreports"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	authhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
)

type Handler struct {
	authenticator     session.Authenticator
	accountRepository accounts.BandAccountRepository
	reportRepository  applicationfinancialreports.Repository
	logger            *slog.Logger
	now               func() time.Time
}

type FinancialReportResponse struct {
	Range          ReportRangeResponse              `json:"range"`
	Summary        ReportSummaryResponse            `json:"summary"`
	PaymentMethods []PaymentMethodBreakdownResponse `json:"paymentMethods"`
	Categories     []CategoryBreakdownResponse      `json:"categories"`
	Products       []ProductBreakdownResponse       `json:"products"`
	Days           []DayBreakdownResponse           `json:"days"`
}

type ReportRangeResponse struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Timezone string `json:"timezone"`
}

type ReportSummaryResponse struct {
	SaleCount           int                 `json:"saleCount"`
	ItemCount           int                 `json:"itemCount"`
	GrossRevenue        MoneyResponse       `json:"grossRevenue"`
	TotalHistoricalCost MoneyResponse       `json:"totalHistoricalCost"`
	ExpectedProfit      SignedMoneyResponse `json:"expectedProfit"`
}

type PaymentMethodBreakdownResponse struct {
	Method              string              `json:"method"`
	SaleCount           int                 `json:"saleCount"`
	ItemCount           int                 `json:"itemCount"`
	GrossRevenue        MoneyResponse       `json:"grossRevenue"`
	TotalHistoricalCost MoneyResponse       `json:"totalHistoricalCost"`
	ExpectedProfit      SignedMoneyResponse `json:"expectedProfit"`
}

type CategoryBreakdownResponse struct {
	Category            string              `json:"category"`
	SaleCount           int                 `json:"saleCount"`
	ItemCount           int                 `json:"itemCount"`
	GrossRevenue        MoneyResponse       `json:"grossRevenue"`
	TotalHistoricalCost MoneyResponse       `json:"totalHistoricalCost"`
	ExpectedProfit      SignedMoneyResponse `json:"expectedProfit"`
}

type ProductBreakdownResponse struct {
	ProductID           string              `json:"productId"`
	ProductName         string              `json:"productName"`
	Category            string              `json:"category"`
	SaleCount           int                 `json:"saleCount"`
	ItemCount           int                 `json:"itemCount"`
	GrossRevenue        MoneyResponse       `json:"grossRevenue"`
	TotalHistoricalCost MoneyResponse       `json:"totalHistoricalCost"`
	ExpectedProfit      SignedMoneyResponse `json:"expectedProfit"`
}

type DayBreakdownResponse struct {
	Date                string              `json:"date"`
	SaleCount           int                 `json:"saleCount"`
	ItemCount           int                 `json:"itemCount"`
	GrossRevenue        MoneyResponse       `json:"grossRevenue"`
	TotalHistoricalCost MoneyResponse       `json:"totalHistoricalCost"`
	ExpectedProfit      SignedMoneyResponse `json:"expectedProfit"`
}

type MoneyResponse struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type SignedMoneyResponse struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

func NewHandler(authenticator session.Authenticator, accountRepository accounts.BandAccountRepository, reportRepository applicationfinancialreports.Repository, logger *slog.Logger) Handler {
	return Handler{
		authenticator:     authenticator,
		accountRepository: accountRepository,
		reportRepository:  reportRepository,
		logger:            logger,
		now:               time.Now,
	}
}

func (handler Handler) GetFinancialReport(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	report, err := applicationfinancialreports.ListReport(request.Context(), handler.reportRepository, applicationfinancialreports.ListReportInput{
		Account: accountContext,
		From:    request.URL.Query().Get("from"),
		To:      request.URL.Query().Get("to"),
		Now:     handler.now().UTC(),
	})
	if err != nil {
		handler.writeFinancialReportError(response, err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toFinancialReportResponse(report))
}

func (handler Handler) accountContext(response http.ResponseWriter, request *http.Request) (applicationfinancialreports.AccountContext, bool) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return applicationfinancialreports.AccountContext{}, false
	}

	account, err := accounts.GetCurrentAccount(request.Context(), handler.accountRepository, accounts.CurrentAccountQuery{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
	})
	if err != nil {
		handler.logger.Warn("financial report account lookup failed", "error", err, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
		handler.writeError(response, http.StatusUnauthorized, "account_not_found", "Authenticated user is not linked to a band account")
		return applicationfinancialreports.AccountContext{}, false
	}

	return applicationfinancialreports.AccountContext{
		UserID:       account.UserID,
		BandID:       account.BandID,
		BandTimezone: account.BandTimezone,
		Role:         account.Role,
	}, true
}

func (handler Handler) authenticate(response http.ResponseWriter, request *http.Request) (session.AuthenticatedUser, bool) {
	token, err := session.NormalizeBearerToken(request.Header.Get("Authorization"))
	if err != nil {
		handler.writeError(response, http.StatusUnauthorized, "invalid_authorization", err.Error())
		return session.AuthenticatedUser{}, false
	}

	authenticatedUser, err := handler.authenticator.Authenticate(request.Context(), token)
	if err != nil {
		handler.writeError(response, http.StatusUnauthorized, "invalid_session", err.Error())
		return session.AuthenticatedUser{}, false
	}

	return authenticatedUser, true
}

func (handler Handler) writeFinancialReportError(response http.ResponseWriter, err error) {
	handler.logger.Warn("financial report request failed", "error", err)

	switch {
	case strings.Contains(err.Error(), "alpha read access denied"):
		handler.writeError(response, http.StatusForbidden, "read_forbidden", err.Error())
	case strings.Contains(err.Error(), "get financial report"):
		handler.writeError(response, http.StatusInternalServerError, "financial_report_failed", err.Error())
	default:
		handler.writeError(response, http.StatusBadRequest, "invalid_financial_report_request", err.Error())
	}
}

func (handler Handler) writeJSON(response http.ResponseWriter, statusCode int, body interface{}) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		handler.logger.Error("financial report response encoding failed", "error", err)
	}
}

func (handler Handler) writeError(response http.ResponseWriter, statusCode int, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	err := json.NewEncoder(response).Encode(authhandler.ErrorResponse{
		Code:    code,
		Message: message,
	})
	if err != nil {
		handler.logger.Error("financial report error response encoding failed", "error", err, "code", code, "status_code", statusCode)
	}
}

func toFinancialReportResponse(report applicationfinancialreports.Report) FinancialReportResponse {
	return FinancialReportResponse{
		Range: ReportRangeResponse{
			From:     report.Range.From,
			To:       report.Range.To,
			Timezone: report.Range.Timezone,
		},
		Summary:        toReportSummaryResponse(report.Summary),
		PaymentMethods: toPaymentMethodBreakdownResponses(report.PaymentMethods),
		Categories:     toCategoryBreakdownResponses(report.Categories),
		Products:       toProductBreakdownResponses(report.Products),
		Days:           toDayBreakdownResponses(report.Days),
	}
}

func toReportSummaryResponse(summary applicationfinancialreports.ReportSummary) ReportSummaryResponse {
	return ReportSummaryResponse{
		SaleCount:           summary.SaleCount,
		ItemCount:           summary.ItemCount,
		GrossRevenue:        toMoneyResponse(summary.GrossRevenue.Amount, summary.GrossRevenue.Currency),
		TotalHistoricalCost: toMoneyResponse(summary.TotalHistoricalCost.Amount, summary.TotalHistoricalCost.Currency),
		ExpectedProfit:      toSignedMoneyResponse(summary.ExpectedProfit.Amount, summary.ExpectedProfit.Currency),
	}
}

func toPaymentMethodBreakdownResponses(breakdowns []applicationfinancialreports.PaymentMethodBreakdown) []PaymentMethodBreakdownResponse {
	responses := make([]PaymentMethodBreakdownResponse, 0, len(breakdowns))
	for _, breakdown := range breakdowns {
		responses = append(responses, PaymentMethodBreakdownResponse{
			Method:              string(breakdown.Method),
			SaleCount:           breakdown.SaleCount,
			ItemCount:           breakdown.ItemCount,
			GrossRevenue:        toMoneyResponse(breakdown.GrossRevenue.Amount, breakdown.GrossRevenue.Currency),
			TotalHistoricalCost: toMoneyResponse(breakdown.TotalHistoricalCost.Amount, breakdown.TotalHistoricalCost.Currency),
			ExpectedProfit:      toSignedMoneyResponse(breakdown.ExpectedProfit.Amount, breakdown.ExpectedProfit.Currency),
		})
	}

	return responses
}

func toCategoryBreakdownResponses(breakdowns []applicationfinancialreports.CategoryBreakdown) []CategoryBreakdownResponse {
	responses := make([]CategoryBreakdownResponse, 0, len(breakdowns))
	for _, breakdown := range breakdowns {
		responses = append(responses, CategoryBreakdownResponse{
			Category:            string(breakdown.Category),
			SaleCount:           breakdown.SaleCount,
			ItemCount:           breakdown.ItemCount,
			GrossRevenue:        toMoneyResponse(breakdown.GrossRevenue.Amount, breakdown.GrossRevenue.Currency),
			TotalHistoricalCost: toMoneyResponse(breakdown.TotalHistoricalCost.Amount, breakdown.TotalHistoricalCost.Currency),
			ExpectedProfit:      toSignedMoneyResponse(breakdown.ExpectedProfit.Amount, breakdown.ExpectedProfit.Currency),
		})
	}

	return responses
}

func toProductBreakdownResponses(breakdowns []applicationfinancialreports.ProductBreakdown) []ProductBreakdownResponse {
	responses := make([]ProductBreakdownResponse, 0, len(breakdowns))
	for _, breakdown := range breakdowns {
		responses = append(responses, ProductBreakdownResponse{
			ProductID:           breakdown.ProductID,
			ProductName:         breakdown.ProductName,
			Category:            string(breakdown.Category),
			SaleCount:           breakdown.SaleCount,
			ItemCount:           breakdown.ItemCount,
			GrossRevenue:        toMoneyResponse(breakdown.GrossRevenue.Amount, breakdown.GrossRevenue.Currency),
			TotalHistoricalCost: toMoneyResponse(breakdown.TotalHistoricalCost.Amount, breakdown.TotalHistoricalCost.Currency),
			ExpectedProfit:      toSignedMoneyResponse(breakdown.ExpectedProfit.Amount, breakdown.ExpectedProfit.Currency),
		})
	}

	return responses
}

func toDayBreakdownResponses(breakdowns []applicationfinancialreports.DayBreakdown) []DayBreakdownResponse {
	responses := make([]DayBreakdownResponse, 0, len(breakdowns))
	for _, breakdown := range breakdowns {
		responses = append(responses, DayBreakdownResponse{
			Date:                breakdown.Date,
			SaleCount:           breakdown.SaleCount,
			ItemCount:           breakdown.ItemCount,
			GrossRevenue:        toMoneyResponse(breakdown.GrossRevenue.Amount, breakdown.GrossRevenue.Currency),
			TotalHistoricalCost: toMoneyResponse(breakdown.TotalHistoricalCost.Amount, breakdown.TotalHistoricalCost.Currency),
			ExpectedProfit:      toSignedMoneyResponse(breakdown.ExpectedProfit.Amount, breakdown.ExpectedProfit.Currency),
		})
	}

	return responses
}

func toMoneyResponse(amount int, currency string) MoneyResponse {
	return MoneyResponse{Amount: amount, Currency: currency}
}

func toSignedMoneyResponse(amount int, currency string) SignedMoneyResponse {
	return SignedMoneyResponse{Amount: amount, Currency: currency}
}

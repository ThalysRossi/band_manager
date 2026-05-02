package merchboothhandler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	authhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Handler struct {
	authenticator     session.Authenticator
	accountRepository accounts.BandAccountRepository
	merchRepository   applicationmerchbooth.Repository
	paymentProvider   applicationmerchbooth.PaymentProvider
	logger            *slog.Logger
	now               func() time.Time
}

type BoothItemsResponse struct {
	Items []BoothItemResponse `json:"items"`
}

type BoothItemResponse struct {
	ProductID   string        `json:"productId"`
	VariantID   string        `json:"variantId"`
	ProductName string        `json:"productName"`
	Category    string        `json:"category"`
	Size        string        `json:"size"`
	Colour      string        `json:"colour"`
	Price       MoneyResponse `json:"price"`
	Cost        MoneyResponse `json:"cost"`
	Quantity    int           `json:"quantity"`
	SoldOut     bool          `json:"soldOut"`
	Photo       PhotoResponse `json:"photo"`
}

type CashCheckoutRequest struct {
	Items []CartItemRequest `json:"items"`
}

type PixCheckoutRequest struct {
	Items []CartItemRequest `json:"items"`
}

type CartItemRequest struct {
	VariantID string `json:"variantId"`
	Quantity  *int   `json:"quantity"`
}

type SaleResponse struct {
	ID             string                `json:"id"`
	BandID         string                `json:"bandId"`
	Status         string                `json:"status"`
	Total          MoneyResponse         `json:"total"`
	ExpectedProfit MoneyResponse         `json:"expectedProfit"`
	Items          []SaleItemResponse    `json:"items"`
	Payment        PaymentResponse       `json:"payment"`
	Transactions   []TransactionResponse `json:"transactions"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}

type SaleItemResponse struct {
	ID             string        `json:"id"`
	SaleID         string        `json:"saleId"`
	ProductID      string        `json:"productId"`
	VariantID      string        `json:"variantId"`
	ProductName    string        `json:"productName"`
	Category       string        `json:"category"`
	Size           string        `json:"size"`
	Colour         string        `json:"colour"`
	Quantity       int           `json:"quantity"`
	UnitPrice      MoneyResponse `json:"unitPrice"`
	UnitCost       MoneyResponse `json:"unitCost"`
	LineTotal      MoneyResponse `json:"lineTotal"`
	ExpectedProfit MoneyResponse `json:"expectedProfit"`
	CreatedAt      time.Time     `json:"createdAt"`
}

type PaymentResponse struct {
	ID                   string        `json:"id"`
	SaleID               string        `json:"saleId"`
	Method               string        `json:"method"`
	Status               string        `json:"status"`
	Amount               MoneyResponse `json:"amount"`
	Provider             string        `json:"provider,omitempty"`
	ProviderOrderID      string        `json:"providerOrderId,omitempty"`
	ProviderPaymentID    string        `json:"providerPaymentId,omitempty"`
	ProviderReferenceID  string        `json:"providerReferenceId,omitempty"`
	ExternalReference    string        `json:"externalReference,omitempty"`
	ProviderStatus       string        `json:"providerStatus,omitempty"`
	ProviderStatusDetail string        `json:"providerStatusDetail,omitempty"`
	ExpiresAt            time.Time     `json:"expiresAt,omitempty"`
	PixQRCode            string        `json:"pixQrCode,omitempty"`
	PixQRCodeBase64      string        `json:"pixQrCodeBase64,omitempty"`
	PixTicketURL         string        `json:"pixTicketUrl,omitempty"`
	CreatedAt            time.Time     `json:"createdAt"`
	UpdatedAt            time.Time     `json:"updatedAt"`
}

type TransactionResponse struct {
	ID         string        `json:"id"`
	SaleID     string        `json:"saleId"`
	SaleItemID string        `json:"saleItemId"`
	Amount     MoneyResponse `json:"amount"`
	CreatedAt  time.Time     `json:"createdAt"`
}

type MoneyResponse struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type PhotoResponse struct {
	ObjectKey   string `json:"objectKey"`
	ContentType string `json:"contentType"`
	SizeBytes   int    `json:"sizeBytes"`
}

func NewHandler(authenticator session.Authenticator, accountRepository accounts.BandAccountRepository, merchRepository applicationmerchbooth.Repository, paymentProvider applicationmerchbooth.PaymentProvider, logger *slog.Logger) Handler {
	return Handler{
		authenticator:     authenticator,
		accountRepository: accountRepository,
		merchRepository:   merchRepository,
		paymentProvider:   paymentProvider,
		logger:            logger,
		now:               time.Now,
	}
}

func (handler Handler) ListBoothItems(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	items, err := applicationmerchbooth.ListBoothItems(request.Context(), handler.merchRepository, applicationmerchbooth.ListBoothItemsInput{
		Account: accountContext,
	})
	if err != nil {
		handler.writeMerchBoothError(response, "merch booth list failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, BoothItemsResponse{Items: toBoothItemResponses(items)})
}

func (handler Handler) CreateCashCheckout(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body CashCheckoutRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	items, ok := toCartItemInputs(response, body.Items)
	if !ok {
		return
	}

	sale, err := applicationmerchbooth.CreateCashCheckout(request.Context(), handler.merchRepository, applicationmerchbooth.CreateCashCheckoutInput{
		Account:        accountContext,
		Items:          items,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeMerchBoothError(response, "cash checkout failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toSaleResponse(sale))
}

func (handler Handler) CreatePixCheckout(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body PixCheckoutRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	items, ok := toCartItemInputs(response, body.Items)
	if !ok {
		return
	}

	sale, err := applicationmerchbooth.CreatePixCheckout(request.Context(), handler.merchRepository, handler.paymentProvider, applicationmerchbooth.CreatePixCheckoutInput{
		Account:        accountContext,
		Items:          items,
		PayerEmail:     accountContext.Email,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeMerchBoothError(response, "pix checkout failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toSaleResponse(sale))
}

func (handler Handler) accountContext(response http.ResponseWriter, request *http.Request) (applicationmerchbooth.AccountContext, bool) {
	authenticatedUser, ok := handler.authenticate(response, request)
	if !ok {
		return applicationmerchbooth.AccountContext{}, false
	}

	account, err := accounts.GetCurrentAccount(request.Context(), handler.accountRepository, accounts.CurrentAccountQuery{
		AuthProvider:       authenticatedUser.Provider,
		AuthProviderUserID: authenticatedUser.ProviderUserID,
	})
	if err != nil {
		handler.logger.Warn("merch booth account lookup failed", "error", err, "provider", authenticatedUser.Provider, "provider_user_id", authenticatedUser.ProviderUserID)
		handler.writeError(response, http.StatusUnauthorized, "account_not_found", "Authenticated user is not linked to a band account")
		return applicationmerchbooth.AccountContext{}, false
	}

	return applicationmerchbooth.AccountContext{
		UserID: account.UserID,
		BandID: account.BandID,
		Email:  account.Email,
		Role:   account.Role,
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

func (handler Handler) mutationHeaders(response http.ResponseWriter, request *http.Request) (string, string, bool) {
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		handler.writeError(response, http.StatusBadRequest, "missing_idempotency_key", "Idempotency-Key header is required")
		return "", "", false
	}

	requestID, ok := middleware.RequestIDFromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_request_id", "request id is missing")
		return "", "", false
	}

	return idempotencyKey, requestID, true
}

func (handler Handler) decodeJSON(response http.ResponseWriter, request *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		handler.writeError(response, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON")
		return false
	}

	return true
}

func (handler Handler) writeMerchBoothError(response http.ResponseWriter, logMessage string, err error) {
	handler.logger.Warn(logMessage, "error", err)

	switch {
	case errors.Is(err, applicationmerchbooth.ErrEmptyCart):
		handler.writeError(response, http.StatusBadRequest, "empty_cart", err.Error())
	case errors.Is(err, applicationmerchbooth.ErrDuplicateCartItem):
		handler.writeError(response, http.StatusBadRequest, "duplicate_cart_item", err.Error())
	case errors.Is(err, applicationmerchbooth.ErrInsufficientStock):
		handler.writeError(response, http.StatusConflict, "insufficient_stock", err.Error())
	case errors.Is(err, applicationmerchbooth.ErrBoothItemNotFound):
		handler.writeError(response, http.StatusNotFound, "booth_item_not_found", err.Error())
	case errors.Is(err, applicationmerchbooth.ErrIdempotencyConflict):
		handler.writeError(response, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, applicationmerchbooth.ErrPaymentProvider):
		handler.writeError(response, http.StatusBadGateway, "payment_provider_failed", err.Error())
	case strings.Contains(err.Error(), "alpha write access requires owner role"):
		handler.writeError(response, http.StatusForbidden, "write_forbidden", err.Error())
	default:
		handler.writeError(response, http.StatusBadRequest, "invalid_merch_booth_request", err.Error())
	}
}

func (handler Handler) writeJSON(response http.ResponseWriter, statusCode int, body interface{}) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		handler.logger.Error("merch booth response encoding failed", "error", err)
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
		handler.logger.Error("merch booth error response encoding failed", "error", err, "code", code, "status_code", statusCode)
	}
}

func toCartItemInputs(response http.ResponseWriter, requests []CartItemRequest) ([]applicationmerchbooth.CartItemInput, bool) {
	items := make([]applicationmerchbooth.CartItemInput, 0, len(requests))
	for _, request := range requests {
		if request.Quantity == nil {
			writeRequestError(response, "missing_quantity", "items.quantity is required")
			return nil, false
		}
		items = append(items, applicationmerchbooth.CartItemInput{
			VariantID: request.VariantID,
			Quantity:  *request.Quantity,
		})
	}

	return items, true
}

func toBoothItemResponses(items []applicationmerchbooth.BoothItem) []BoothItemResponse {
	responses := make([]BoothItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, BoothItemResponse{
			ProductID:   item.ProductID,
			VariantID:   item.VariantID,
			ProductName: item.ProductName,
			Category:    string(item.Category),
			Size:        string(item.Size),
			Colour:      item.Colour,
			Price:       toMoneyResponse(item.Price.Amount, item.Price.Currency),
			Cost:        toMoneyResponse(item.Cost.Amount, item.Cost.Currency),
			Quantity:    item.Quantity,
			SoldOut:     item.SoldOut,
			Photo: PhotoResponse{
				ObjectKey:   item.Photo.ObjectKey,
				ContentType: item.Photo.ContentType,
				SizeBytes:   item.Photo.SizeBytes,
			},
		})
	}

	return responses
}

func toSaleResponse(sale applicationmerchbooth.Sale) SaleResponse {
	return SaleResponse{
		ID:             sale.ID,
		BandID:         sale.BandID,
		Status:         string(sale.Status),
		Total:          toMoneyResponse(sale.Total.Amount, sale.Total.Currency),
		ExpectedProfit: toMoneyResponse(sale.ExpectedProfit.Amount, sale.ExpectedProfit.Currency),
		Items:          toSaleItemResponses(sale.Items),
		Payment:        toPaymentResponse(sale.Payment),
		Transactions:   toTransactionResponses(sale.Transactions),
		CreatedAt:      sale.CreatedAt,
		UpdatedAt:      sale.UpdatedAt,
	}
}

func toSaleItemResponses(items []applicationmerchbooth.SaleItem) []SaleItemResponse {
	responses := make([]SaleItemResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, SaleItemResponse{
			ID:             item.ID,
			SaleID:         item.SaleID,
			ProductID:      item.ProductID,
			VariantID:      item.VariantID,
			ProductName:    item.ProductName,
			Category:       string(item.Category),
			Size:           string(item.Size),
			Colour:         item.Colour,
			Quantity:       item.Quantity,
			UnitPrice:      toMoneyResponse(item.UnitPrice.Amount, item.UnitPrice.Currency),
			UnitCost:       toMoneyResponse(item.UnitCost.Amount, item.UnitCost.Currency),
			LineTotal:      toMoneyResponse(item.LineTotal.Amount, item.LineTotal.Currency),
			ExpectedProfit: toMoneyResponse(item.ExpectedProfit.Amount, item.ExpectedProfit.Currency),
			CreatedAt:      item.CreatedAt,
		})
	}

	return responses
}

func toPaymentResponse(payment applicationmerchbooth.Payment) PaymentResponse {
	return PaymentResponse{
		ID:                   payment.ID,
		SaleID:               payment.SaleID,
		Method:               string(payment.Method),
		Status:               string(payment.Status),
		Amount:               toMoneyResponse(payment.Amount.Amount, payment.Amount.Currency),
		Provider:             payment.Provider,
		ProviderOrderID:      payment.ProviderOrderID,
		ProviderPaymentID:    payment.ProviderPaymentID,
		ProviderReferenceID:  payment.ProviderReferenceID,
		ExternalReference:    payment.ExternalReference,
		ProviderStatus:       payment.ProviderStatus,
		ProviderStatusDetail: payment.ProviderStatusDetail,
		ExpiresAt:            payment.ExpiresAt,
		PixQRCode:            payment.PixQRCode,
		PixQRCodeBase64:      payment.PixQRCodeBase64,
		PixTicketURL:         payment.PixTicketURL,
		CreatedAt:            payment.CreatedAt,
		UpdatedAt:            payment.UpdatedAt,
	}
}

func toTransactionResponses(transactions []applicationmerchbooth.Transaction) []TransactionResponse {
	responses := make([]TransactionResponse, 0, len(transactions))
	for _, transaction := range transactions {
		responses = append(responses, TransactionResponse{
			ID:         transaction.ID,
			SaleID:     transaction.SaleID,
			SaleItemID: transaction.SaleItemID,
			Amount:     toMoneyResponse(transaction.Amount.Amount, transaction.Amount.Currency),
			CreatedAt:  transaction.CreatedAt,
		})
	}

	return responses
}

func toMoneyResponse(amount int, currency string) MoneyResponse {
	return MoneyResponse{Amount: amount, Currency: currency}
}

func writeRequestError(response http.ResponseWriter, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(response).Encode(authhandler.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

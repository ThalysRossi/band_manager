package mercadopago

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
)

const ProviderName = "mercadopago"

type Client struct {
	accessToken string
	baseURL     string
	httpClient  *http.Client
	logger      *slog.Logger
}

type orderRequest struct {
	Type              string            `json:"type"`
	TotalAmount       string            `json:"total_amount"`
	ExternalReference string            `json:"external_reference"`
	ExpirationTime    string            `json:"expiration_time,omitempty"`
	ProcessingMode    string            `json:"processing_mode"`
	Config            *orderConfig      `json:"config,omitempty"`
	Transactions      orderTransactions `json:"transactions"`
	Payer             *orderPayer       `json:"payer,omitempty"`
}

type orderTransactions struct {
	Payments []orderPaymentRequest `json:"payments"`
}

type orderPaymentRequest struct {
	Amount         string              `json:"amount"`
	PaymentMethod  *orderPaymentMethod `json:"payment_method,omitempty"`
	ExpirationTime string              `json:"expiration_time,omitempty"`
}

type orderPaymentMethod struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type orderPayer struct {
	Email string `json:"email"`
}

type orderConfig struct {
	Point         orderPointConfig         `json:"point"`
	PaymentMethod orderPaymentMethodConfig `json:"payment_method"`
}

type orderPointConfig struct {
	TerminalID      string `json:"terminal_id"`
	PrintOnTerminal string `json:"print_on_terminal"`
}

type orderPaymentMethodConfig struct {
	DefaultType         string `json:"default_type"`
	DefaultInstallments int    `json:"default_installments"`
	InstallmentsCost    string `json:"installments_cost"`
}

type orderResponse struct {
	ID                string                   `json:"id"`
	ExternalReference string                   `json:"external_reference"`
	Status            string                   `json:"status"`
	StatusDetail      string                   `json:"status_detail"`
	Transactions      orderResponseTransaction `json:"transactions"`
}

type orderResponseTransaction struct {
	Payments []paymentResponse `json:"payments"`
}

type paymentResponse struct {
	ID              string          `json:"id"`
	ReferenceID     string          `json:"reference_id"`
	Status          string          `json:"status"`
	StatusDetail    string          `json:"status_detail"`
	RawPaymentValue json.RawMessage `json:"-"`
}

func NewClient(accessToken string, baseURL string, httpClient *http.Client, logger *slog.Logger) (Client, error) {
	trimmedAccessToken := strings.TrimSpace(accessToken)
	if trimmedAccessToken == "" {
		return Client{}, fmt.Errorf("mercadopago access token is required")
	}

	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBaseURL == "" {
		return Client{}, fmt.Errorf("mercadopago base url is required")
	}

	if httpClient == nil {
		return Client{}, fmt.Errorf("http client is required")
	}

	if logger == nil {
		return Client{}, fmt.Errorf("logger is required")
	}

	return Client{
		accessToken: trimmedAccessToken,
		baseURL:     trimmedBaseURL,
		httpClient:  httpClient,
		logger:      logger,
	}, nil
}

func (client Client) CreatePixPayment(ctx context.Context, command applicationmerchbooth.CreatePixPaymentCommand) (applicationmerchbooth.PixPayment, error) {
	requestBody := orderRequest{
		Type:              "online",
		TotalAmount:       minorUnitsToDecimal(command.Amount),
		ExternalReference: command.ExternalReference,
		ProcessingMode:    "automatic",
		Transactions: orderTransactions{
			Payments: []orderPaymentRequest{
				{
					Amount: minorUnitsToDecimal(command.Amount),
					PaymentMethod: &orderPaymentMethod{
						ID:   "pix",
						Type: "bank_transfer",
					},
					ExpirationTime: "PT30M",
				},
			},
		},
		Payer: &orderPayer{Email: command.PayerEmail},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("marshal mercadopago pix order request external_reference=%q: %w", command.ExternalReference, err)
	}

	responseBody, err := client.doOrderRequest(ctx, body, command.IdempotencyKey, command.ExternalReference)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, err
	}

	var order orderResponse
	if err := json.Unmarshal(responseBody, &order); err != nil {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("decode mercadopago pix order response external_reference=%q response_body=%q: %w", command.ExternalReference, string(responseBody), err)
	}

	if order.ID == "" {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("mercadopago pix order response missing order id external_reference=%q response_body=%q", command.ExternalReference, string(responseBody))
	}

	if len(order.Transactions.Payments) == 0 {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("mercadopago pix order response missing payment external_reference=%q order_id=%q response_body=%q", command.ExternalReference, order.ID, string(responseBody))
	}

	payment := order.Transactions.Payments[0]
	qrCode := findStringValue(responseBody, "qr_code")
	qrCodeBase64 := findStringValue(responseBody, "qr_code_base64")
	ticketURL := findStringValue(responseBody, "ticket_url")

	if qrCode == "" || qrCodeBase64 == "" {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("mercadopago pix order response missing qr fields external_reference=%q order_id=%q response_body=%q", command.ExternalReference, order.ID, string(responseBody))
	}

	localStatus, err := localStatusForProviderStatus(order.Status)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("map mercadopago pix order status external_reference=%q order_id=%q status=%q response_body=%q: %w", command.ExternalReference, order.ID, order.Status, string(responseBody), err)
	}

	return applicationmerchbooth.PixPayment{
		Provider:             ProviderName,
		ProviderOrderID:      order.ID,
		ProviderPaymentID:    payment.ID,
		ProviderReferenceID:  payment.ReferenceID,
		ExternalReference:    command.ExternalReference,
		ProviderStatus:       order.Status,
		ProviderStatusDetail: firstNonEmpty(payment.StatusDetail, order.StatusDetail),
		LocalStatus:          localStatus,
		Amount:               command.Amount,
		ExpiresAt:            command.ExpiresAt,
		QRCode:               qrCode,
		QRCodeBase64:         qrCodeBase64,
		TicketURL:            ticketURL,
		RawProviderResponse:  responseBody,
	}, nil
}

func (client Client) CreateCardPayment(ctx context.Context, command applicationmerchbooth.CreateCardPaymentCommand) (applicationmerchbooth.PixPayment, error) {
	requestBody := orderRequest{
		Type:              "point",
		TotalAmount:       minorUnitsToDecimal(command.Amount),
		ExternalReference: command.ExternalReference,
		ExpirationTime:    "PT16M",
		ProcessingMode:    "automatic",
		Config: &orderConfig{
			Point: orderPointConfig{
				TerminalID:      command.TerminalID,
				PrintOnTerminal: "no_ticket",
			},
			PaymentMethod: orderPaymentMethodConfig{
				DefaultType:         string(command.CardType),
				DefaultInstallments: command.Installments,
				InstallmentsCost:    "seller",
			},
		},
		Transactions: orderTransactions{
			Payments: []orderPaymentRequest{
				{Amount: minorUnitsToDecimal(command.Amount)},
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("marshal mercadopago card order request external_reference=%q terminal_id=%q: %w", command.ExternalReference, command.TerminalID, err)
	}

	responseBody, err := client.doOrderRequest(ctx, body, command.IdempotencyKey, command.ExternalReference)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, err
	}

	payment, err := pixPaymentFromOrderResponse(responseBody, command.ExternalReference, command.Amount)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, err
	}
	payment.ExpiresAt = command.ExpiresAt

	return payment, nil
}

func (client Client) GetPaymentStatus(ctx context.Context, command applicationmerchbooth.GetPaymentStatusCommand) (applicationmerchbooth.PixPayment, error) {
	providerOrderID := strings.TrimSpace(command.ProviderOrderID)
	if providerOrderID == "" {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("mercadopago order id is required")
	}

	responseBody, err := client.doGetOrderRequest(ctx, providerOrderID)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, err
	}

	payment, err := pixPaymentFromOrderResponse(responseBody, providerOrderID, inventorydomain.Money{})
	if err != nil {
		return applicationmerchbooth.PixPayment{}, err
	}

	return payment, nil
}

func (client Client) doOrderRequest(ctx context.Context, body []byte, idempotencyKey string, externalReference string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/v1/orders", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create mercadopago pix order request external_reference=%q: %w", externalReference, err)
		}
		request.Header.Set("Authorization", "Bearer "+client.accessToken)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Idempotency-Key", idempotencyKey)

		response, err := client.httpClient.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("mercadopago pix order request failed external_reference=%q attempt=%d: %w", externalReference, attempt, err)
			client.logger.Warn("mercadopago pix order request failed", "error", err, "provider", ProviderName, "operation", "create_pix_order", "external_reference", externalReference, "attempt", attempt)
			continue
		}

		responseBody, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read mercadopago pix order response external_reference=%q status_code=%d attempt=%d: %w", externalReference, response.StatusCode, attempt, readErr)
			continue
		}
		if closeErr != nil {
			lastErr = fmt.Errorf("close mercadopago pix order response external_reference=%q status_code=%d attempt=%d: %w", externalReference, response.StatusCode, attempt, closeErr)
			continue
		}

		if response.StatusCode >= 200 && response.StatusCode <= 299 {
			return responseBody, nil
		}

		lastErr = fmt.Errorf("mercadopago pix order response failed external_reference=%q status_code=%d response_body=%q", externalReference, response.StatusCode, string(responseBody))
		if response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
			return nil, lastErr
		}

		client.logger.Warn("mercadopago pix order response retryable failure", "provider", ProviderName, "operation", "create_pix_order", "external_reference", externalReference, "status_code", response.StatusCode, "attempt", attempt)
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return nil, lastErr
}

func (client Client) doGetOrderRequest(ctx context.Context, providerOrderID string) ([]byte, error) {
	var lastErr error
	escapedOrderID := url.PathEscape(providerOrderID)
	for attempt := 1; attempt <= 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/v1/orders/"+escapedOrderID, nil)
		if err != nil {
			return nil, fmt.Errorf("create mercadopago get order request provider_order_id=%q: %w", providerOrderID, err)
		}
		request.Header.Set("Authorization", "Bearer "+client.accessToken)
		request.Header.Set("Content-Type", "application/json")

		response, err := client.httpClient.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("mercadopago get order request failed provider_order_id=%q attempt=%d: %w", providerOrderID, attempt, err)
			client.logger.Warn("mercadopago get order request failed", "error", err, "provider", ProviderName, "operation", "get_order", "provider_order_id", providerOrderID, "attempt", attempt)
			continue
		}

		responseBody, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read mercadopago get order response provider_order_id=%q status_code=%d attempt=%d: %w", providerOrderID, response.StatusCode, attempt, readErr)
			continue
		}
		if closeErr != nil {
			lastErr = fmt.Errorf("close mercadopago get order response provider_order_id=%q status_code=%d attempt=%d: %w", providerOrderID, response.StatusCode, attempt, closeErr)
			continue
		}

		if response.StatusCode >= 200 && response.StatusCode <= 299 {
			return responseBody, nil
		}

		lastErr = fmt.Errorf("mercadopago get order response failed provider_order_id=%q status_code=%d response_body=%q", providerOrderID, response.StatusCode, string(responseBody))
		if response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
			return nil, lastErr
		}

		client.logger.Warn("mercadopago get order response retryable failure", "provider", ProviderName, "operation", "get_order", "provider_order_id", providerOrderID, "status_code", response.StatusCode, "attempt", attempt)
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return nil, lastErr
}

func pixPaymentFromOrderResponse(responseBody []byte, externalReference string, amount inventorydomain.Money) (applicationmerchbooth.PixPayment, error) {
	var order orderResponse
	if err := json.Unmarshal(responseBody, &order); err != nil {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("decode mercadopago order response external_reference=%q response_body=%q: %w", externalReference, string(responseBody), err)
	}

	if order.ID == "" {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("mercadopago order response missing order id external_reference=%q response_body=%q", externalReference, string(responseBody))
	}

	localStatus, err := localStatusForProviderStatus(order.Status)
	if err != nil {
		return applicationmerchbooth.PixPayment{}, fmt.Errorf("map mercadopago order status external_reference=%q order_id=%q status=%q response_body=%q: %w", externalReference, order.ID, order.Status, string(responseBody), err)
	}

	payment := paymentResponse{}
	if len(order.Transactions.Payments) > 0 {
		payment = order.Transactions.Payments[0]
	}

	return applicationmerchbooth.PixPayment{
		Provider:             ProviderName,
		ProviderOrderID:      order.ID,
		ProviderPaymentID:    payment.ID,
		ProviderReferenceID:  payment.ReferenceID,
		ExternalReference:    firstNonEmpty(order.ExternalReference, externalReference),
		ProviderStatus:       order.Status,
		ProviderStatusDetail: firstNonEmpty(payment.StatusDetail, order.StatusDetail),
		LocalStatus:          localStatus,
		Amount:               amount,
		QRCode:               findStringValue(responseBody, "qr_code"),
		QRCodeBase64:         findStringValue(responseBody, "qr_code_base64"),
		TicketURL:            findStringValue(responseBody, "ticket_url"),
		RawProviderResponse:  responseBody,
	}, nil
}

func minorUnitsToDecimal(money inventorydomain.Money) string {
	reais := money.Amount / 100
	centavos := money.Amount % 100
	return fmt.Sprintf("%d.%02d", reais, centavos)
}

func localStatusForProviderStatus(status string) (applicationmerchbooth.PaymentStatus, error) {
	switch status {
	case "created":
		return applicationmerchbooth.PaymentStatusProviderPending, nil
	case "action_required":
		return applicationmerchbooth.PaymentStatusActionRequired, nil
	case "processing", "at_terminal":
		return applicationmerchbooth.PaymentStatusProcessing, nil
	case "processed":
		return applicationmerchbooth.PaymentStatusConfirmed, nil
	case "failed":
		return applicationmerchbooth.PaymentStatusFailed, nil
	case "canceled":
		return applicationmerchbooth.PaymentStatusCanceled, nil
	case "expired":
		return applicationmerchbooth.PaymentStatusExpired, nil
	default:
		return "", fmt.Errorf("unsupported mercadopago order status %q", status)
	}
}

func findStringValue(body []byte, key string) string {
	var value interface{}
	if err := json.Unmarshal(body, &value); err != nil {
		return ""
	}

	return findStringValueInInterface(value, key)
}

func findStringValueInInterface(value interface{}, key string) string {
	switch typedValue := value.(type) {
	case map[string]interface{}:
		for mapKey, mapValue := range typedValue {
			if mapKey == key {
				stringValue, ok := mapValue.(string)
				if ok {
					return stringValue
				}
			}
			foundValue := findStringValueInInterface(mapValue, key)
			if foundValue != "" {
				return foundValue
			}
		}
	case []interface{}:
		for _, item := range typedValue {
			foundValue := findStringValueInInterface(item, key)
			if foundValue != "" {
				return foundValue
			}
		}
	}

	return ""
}

func firstNonEmpty(first string, second string) string {
	if first != "" {
		return first
	}

	return second
}

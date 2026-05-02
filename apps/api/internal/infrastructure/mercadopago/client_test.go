package mercadopago

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
)

func TestCreatePixPaymentSendsOrderRequestAndParsesQRCode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("expected POST method, got %s", request.Method)
		}
		if request.URL.Path != "/v1/orders" {
			t.Fatalf("expected /v1/orders path, got %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("expected authorization header, got %q", request.Header.Get("Authorization"))
		}
		if request.Header.Get("X-Idempotency-Key") != "idem_1" {
			t.Fatalf("expected idempotency key header, got %q", request.Header.Get("X-Idempotency-Key"))
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected json content type, got %q", request.Header.Get("Content-Type"))
		}

		var body orderRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if body.Type != "online" {
			t.Fatalf("expected online order type, got %q", body.Type)
		}
		if body.TotalAmount != "42.00" {
			t.Fatalf("expected decimal amount, got %q", body.TotalAmount)
		}
		if body.ExternalReference != "sale_1" {
			t.Fatalf("expected external reference, got %q", body.ExternalReference)
		}
		if body.Transactions.Payments[0].PaymentMethod.ID != "pix" {
			t.Fatalf("expected pix payment method, got %q", body.Transactions.Payments[0].PaymentMethod.ID)
		}
		if body.Transactions.Payments[0].PaymentMethod.Type != "bank_transfer" {
			t.Fatalf("expected bank transfer payment method, got %q", body.Transactions.Payments[0].PaymentMethod.Type)
		}
		if body.Transactions.Payments[0].ExpirationTime != "PT30M" {
			t.Fatalf("expected 30 minute expiration, got %q", body.Transactions.Payments[0].ExpirationTime)
		}
		if body.Payer.Email != "band@example.com" {
			t.Fatalf("expected payer email, got %q", body.Payer.Email)
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		response.Write([]byte(`{
			"id": "order_1",
			"status": "action_required",
			"status_detail": "waiting_transfer",
			"transactions": {
				"payments": [
					{
						"id": "payment_1",
						"reference_id": "reference_1",
						"status": "action_required",
						"status_detail": "waiting_transfer",
						"payment_method": {
							"qr_code": "pix-copy-paste",
							"qr_code_base64": "base64",
							"ticket_url": "https://example.test/ticket"
						}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient("access-token", server.URL, server.Client(), slog.Default())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	payment, err := client.CreatePixPayment(context.Background(), applicationmerchbooth.CreatePixPaymentCommand{
		SaleID:            "sale_1",
		PaymentID:         "payment_local_1",
		ExternalReference: "sale_1",
		Amount:            inventorydomain.Money{Amount: 4200, Currency: "BRL"},
		PayerEmail:        "band@example.com",
		IdempotencyKey:    "idem_1",
		ExpiresAt:         time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create pix payment: %v", err)
	}

	if payment.ProviderOrderID != "order_1" {
		t.Fatalf("expected provider order id, got %q", payment.ProviderOrderID)
	}
	if payment.QRCode != "pix-copy-paste" {
		t.Fatalf("expected qr code, got %q", payment.QRCode)
	}
	if payment.LocalStatus != applicationmerchbooth.PaymentStatusActionRequired {
		t.Fatalf("expected action required status, got %q", payment.LocalStatus)
	}
}

func TestGetPaymentStatusFetchesOrderAndMapsStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Fatalf("expected GET method, got %s", request.Method)
		}
		if request.URL.Path != "/v1/orders/order_1" {
			t.Fatalf("expected order path, got %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("expected authorization header, got %q", request.Header.Get("Authorization"))
		}

		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		response.Write([]byte(`{
			"id": "order_1",
			"external_reference": "sale_1",
			"status": "processed",
			"status_detail": "accredited",
			"transactions": {
				"payments": [
					{
						"id": "payment_1",
						"reference_id": "reference_1",
						"status": "processed",
						"status_detail": "accredited"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewClient("access-token", server.URL, server.Client(), slog.Default())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	payment, err := client.GetPaymentStatus(context.Background(), applicationmerchbooth.GetPaymentStatusCommand{ProviderOrderID: "order_1"})
	if err != nil {
		t.Fatalf("get payment status: %v", err)
	}

	if payment.LocalStatus != applicationmerchbooth.PaymentStatusConfirmed {
		t.Fatalf("expected confirmed status, got %q", payment.LocalStatus)
	}
	if payment.ProviderPaymentID != "payment_1" {
		t.Fatalf("expected provider payment id, got %q", payment.ProviderPaymentID)
	}
	if payment.ExternalReference != "sale_1" {
		t.Fatalf("expected external reference, got %q", payment.ExternalReference)
	}
}

package merchbooth

import (
	"context"
	"errors"
	"testing"
	"time"

	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestCreateCashCheckoutRejectsEmptyCart(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateCashCheckoutInput()
	input.Items = []CartItemInput{}

	_, err := CreateCashCheckout(context.Background(), &repository, input)
	if !errors.Is(err, ErrEmptyCart) {
		t.Fatalf("expected empty cart error, got %v", err)
	}
}

func TestCreateCashCheckoutRejectsDuplicateVariantLines(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateCashCheckoutInput()
	input.Items = append(input.Items, input.Items[0])

	_, err := CreateCashCheckout(context.Background(), &repository, input)
	if !errors.Is(err, ErrDuplicateCartItem) {
		t.Fatalf("expected duplicate cart item error, got %v", err)
	}
}

func TestCreateCashCheckoutRequiresOwnerWriteAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateCashCheckoutInput()
	input.Account.Role = permissions.RoleViewer

	_, err := CreateCashCheckout(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected write access error")
	}
}

func TestCreateCashCheckoutStoresSortedCommand(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateCashCheckoutInput()
	input.Items = []CartItemInput{
		{VariantID: "f4814d4c-f402-40a3-937b-e90b0c558222", Quantity: 1},
		{VariantID: "a6ab9f32-6f79-4dec-b232-219d10e75f13", Quantity: 2},
	}

	_, err := CreateCashCheckout(context.Background(), &repository, input)
	if err != nil {
		t.Fatalf("create cash checkout: %v", err)
	}

	if repository.command.Items[0].VariantID != "a6ab9f32-6f79-4dec-b232-219d10e75f13" {
		t.Fatalf("expected sorted cart items, got first variant %q", repository.command.Items[0].VariantID)
	}
}

func TestListBoothItemsAllowsViewerReadAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := ListBoothItemsInput{
		Account: AccountContext{
			UserID: "user_1",
			BandID: "band_1",
			Role:   permissions.RoleViewer,
		},
	}

	_, err := ListBoothItems(context.Background(), &repository, input)
	if err != nil {
		t.Fatalf("list booth items: %v", err)
	}
}

func TestCreatePixCheckoutCallsPaymentProvider(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{
		sale: Sale{
			ID:    "sale_1",
			Total: inventoryMoney(10000),
		},
	}
	provider := fakePaymentProvider{
		payment: PixPayment{
			Provider:            "mercadopago",
			ProviderOrderID:     "order_1",
			ProviderPaymentID:   "payment_1",
			ExternalReference:   "sale_1",
			LocalStatus:         PaymentStatusActionRequired,
			Amount:              inventoryMoney(10000),
			ExpiresAt:           time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC),
			QRCode:              "pix-copy-paste",
			QRCodeBase64:        "base64",
			RawProviderResponse: []byte(`{"id":"order_1"}`),
		},
	}
	input := CreatePixCheckoutInput{
		Account:        validCreateCashCheckoutInput().Account,
		Items:          validCreateCashCheckoutInput().Items,
		PayerEmail:     "band@example.com",
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		CreatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	_, err := CreatePixCheckout(context.Background(), &repository, &provider, input)
	if err != nil {
		t.Fatalf("create pix checkout: %v", err)
	}

	if provider.command.PayerEmail != "band@example.com" {
		t.Fatalf("expected payer email, got %q", provider.command.PayerEmail)
	}

	if repository.completeCommand.ProviderResult.ProviderOrderID != "order_1" {
		t.Fatalf("expected provider order id, got %q", repository.completeCommand.ProviderResult.ProviderOrderID)
	}
}

func TestCreatePixCheckoutReleasesReservationWhenProviderFails(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{
		sale: Sale{
			ID:    "sale_1",
			Total: inventoryMoney(10000),
		},
	}
	provider := fakePaymentProvider{err: errors.New("provider unavailable")}
	input := CreatePixCheckoutInput{
		Account:        validCreateCashCheckoutInput().Account,
		Items:          validCreateCashCheckoutInput().Items,
		PayerEmail:     "band@example.com",
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		CreatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	_, err := CreatePixCheckout(context.Background(), &repository, &provider, input)
	if !errors.Is(err, ErrPaymentProvider) {
		t.Fatalf("expected payment provider error, got %v", err)
	}

	if repository.failCommand.SaleID == "" {
		t.Fatal("expected failed provider checkout to release reservation")
	}
}

func validCreateCashCheckoutInput() CreateCashCheckoutInput {
	return CreateCashCheckoutInput{
		Account: AccountContext{
			UserID: "user_1",
			BandID: "band_1",
			Role:   permissions.RoleOwner,
		},
		Items: []CartItemInput{
			{
				VariantID: "a6ab9f32-6f79-4dec-b232-219d10e75f13",
				Quantity:  2,
			},
		},
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		CreatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
}

type fakeRepository struct {
	command         CreateCashCheckoutCommand
	pixCommand      CreatePixCheckoutCommand
	completeCommand CompletePixCheckoutPaymentCommand
	failCommand     FailPixCheckoutPaymentCreationCommand
	sale            Sale
	found           bool
	err             error
}

func (repository *fakeRepository) ListBoothItems(ctx context.Context, query ListBoothItemsQuery) ([]BoothItem, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}

	return nil, repository.err
}

func (repository *fakeRepository) CreateCashCheckout(ctx context.Context, command CreateCashCheckoutCommand) (Sale, error) {
	if ctx == nil {
		return Sale{}, errors.New("context is required")
	}

	repository.command = command
	return Sale{}, repository.err
}

func (repository *fakeRepository) ReservePixCheckout(ctx context.Context, command CreatePixCheckoutCommand) (Sale, bool, error) {
	if ctx == nil {
		return Sale{}, false, errors.New("context is required")
	}

	repository.pixCommand = command
	return repository.sale, repository.found, repository.err
}

func (repository *fakeRepository) CompletePixCheckoutPayment(ctx context.Context, command CompletePixCheckoutPaymentCommand) (Sale, error) {
	if ctx == nil {
		return Sale{}, errors.New("context is required")
	}

	repository.completeCommand = command
	return repository.sale, repository.err
}

func (repository *fakeRepository) FailPixCheckoutPaymentCreation(ctx context.Context, command FailPixCheckoutPaymentCreationCommand) error {
	if ctx == nil {
		return errors.New("context is required")
	}

	repository.failCommand = command
	return repository.err
}

type fakePaymentProvider struct {
	command CreatePixPaymentCommand
	payment PixPayment
	err     error
}

func (provider *fakePaymentProvider) CreatePixPayment(ctx context.Context, command CreatePixPaymentCommand) (PixPayment, error) {
	if ctx == nil {
		return PixPayment{}, errors.New("context is required")
	}

	provider.command = command
	if provider.err != nil {
		return PixPayment{}, provider.err
	}

	return provider.payment, nil
}

func inventoryMoney(amount int) inventorydomain.Money {
	return inventorydomain.Money{Amount: amount, Currency: "BRL"}
}

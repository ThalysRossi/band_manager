package merchbooth

import (
	"context"
	"errors"
	"testing"
	"time"

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
	command CreateCashCheckoutCommand
	err     error
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

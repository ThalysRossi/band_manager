package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

func TestCreateProductRequiresOwnerWriteAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateProductInput()
	input.Account.Role = permissions.RoleViewer

	_, err := CreateProduct(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected role validation error")
	}
}

func TestCreateProductRejectsDuplicateVariants(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := validCreateProductInput()
	input.Variants = append(input.Variants, VariantInput{
		Size:        "m",
		Colour:      "black",
		PriceAmount: 5000,
		CostAmount:  2000,
		Currency:    "BRL",
		Quantity:    1,
	})

	_, err := CreateProduct(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected duplicate variant validation error")
	}
}

func TestCreateProductStoresValidatedCommand(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{
		product: Product{ID: "product_1"},
	}
	input := validCreateProductInput()

	_, err := CreateProduct(context.Background(), &repository, input)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	if repository.createCommand.NormalizedName != "camiseta logo" {
		t.Fatalf("expected normalized name, got %q", repository.createCommand.NormalizedName)
	}

	if repository.createCommand.Variants[0].NormalizedColour != "black" {
		t.Fatalf("expected normalized colour, got %q", repository.createCommand.Variants[0].NormalizedColour)
	}

	if repository.createCommand.CreatedAt.Location() != time.UTC {
		t.Fatalf("expected UTC created at, got %s", repository.createCommand.CreatedAt.Location())
	}
}

func TestUpdateVariantRejectsNegativeQuantity(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{}
	input := UpdateVariantInput{
		Account:   validAccountContext(),
		VariantID: "variant_1",
		Variant: VariantInput{
			Size:        "m",
			Colour:      "black",
			PriceAmount: 5000,
			CostAmount:  2000,
			Currency:    "BRL",
			Quantity:    -1,
		},
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		UpdatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	_, err := UpdateVariant(context.Background(), &repository, input)
	if err == nil {
		t.Fatal("expected quantity validation error")
	}
}

func TestListInventoryAllowsViewerReadAccess(t *testing.T) {
	t.Parallel()

	repository := fakeRepository{
		products: []Product{
			{
				ID:       "product_1",
				Category: inventorydomain.CategoryShirt,
				Variants: []Variant{
					{ID: "variant_1", Quantity: 0},
				},
			},
		},
	}
	input := ListInventoryInput{
		Account: AccountContext{
			UserID: "user_1",
			BandID: "band_1",
			Role:   permissions.RoleViewer,
		},
	}

	products, err := ListInventory(context.Background(), &repository, input)
	if err != nil {
		t.Fatalf("list inventory: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf("expected products to include sold-out variant, got %d", len(products))
	}
}

func validCreateProductInput() CreateProductInput {
	return CreateProductInput{
		Account:  validAccountContext(),
		Name:     " Camiseta   Logo ",
		Category: "shirt",
		Photo: PhotoInput{
			ObjectKey:   "bands/band_1/products/photo.jpg",
			ContentType: "image/jpeg",
			SizeBytes:   1200,
		},
		Variants: []VariantInput{
			{
				Size:        "m",
				Colour:      " Black ",
				PriceAmount: 5000,
				CostAmount:  2000,
				Currency:    "BRL",
				Quantity:    2,
			},
		},
		IdempotencyKey: "idem_1",
		RequestID:      "request_1",
		CreatedAt:      time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("BRT", -3*60*60)),
	}
}

func validAccountContext() AccountContext {
	return AccountContext{
		UserID: "user_1",
		BandID: "band_1",
		Role:   permissions.RoleOwner,
	}
}

type fakeRepository struct {
	product       Product
	products      []Product
	createCommand CreateProductCommand
	err           error
}

func (repository *fakeRepository) CreateProduct(ctx context.Context, command CreateProductCommand) (Product, error) {
	if ctx == nil {
		return Product{}, errors.New("context is required")
	}

	repository.createCommand = command
	if repository.err != nil {
		return Product{}, repository.err
	}

	return repository.product, nil
}

func (repository *fakeRepository) ListInventory(ctx context.Context, query ListInventoryQuery) ([]Product, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}

	if repository.err != nil {
		return nil, repository.err
	}

	return repository.products, nil
}

func (repository *fakeRepository) UpdateProduct(ctx context.Context, command UpdateProductCommand) (Product, error) {
	if ctx == nil {
		return Product{}, errors.New("context is required")
	}

	if repository.err != nil {
		return Product{}, repository.err
	}

	return repository.product, nil
}

func (repository *fakeRepository) UpdateVariant(ctx context.Context, command UpdateVariantCommand) (Variant, error) {
	if ctx == nil {
		return Variant{}, errors.New("context is required")
	}

	if repository.err != nil {
		return Variant{}, repository.err
	}

	return Variant{}, nil
}

func (repository *fakeRepository) SoftDeleteProduct(ctx context.Context, command SoftDeleteProductCommand) error {
	if ctx == nil {
		return errors.New("context is required")
	}

	return repository.err
}

func (repository *fakeRepository) SoftDeleteVariant(ctx context.Context, command SoftDeleteVariantCommand) error {
	if ctx == nil {
		return errors.New("context is required")
	}

	return repository.err
}

package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

var (
	ErrDuplicateProduct  = errors.New("duplicate inventory product")
	ErrDuplicateVariant  = errors.New("duplicate inventory variant")
	ErrInventoryNotFound = errors.New("inventory record not found")
)

type Repository interface {
	CreateProduct(ctx context.Context, command CreateProductCommand) (Product, error)
	ListInventory(ctx context.Context, query ListInventoryQuery) ([]Product, error)
	UpdateProduct(ctx context.Context, command UpdateProductCommand) (Product, error)
	UpdateVariant(ctx context.Context, command UpdateVariantCommand) (Variant, error)
	SoftDeleteProduct(ctx context.Context, command SoftDeleteProductCommand) error
	SoftDeleteVariant(ctx context.Context, command SoftDeleteVariantCommand) error
}

type AccountContext struct {
	UserID string
	BandID string
	Role   permissions.Role
}

type MoneyInput struct {
	Amount   int
	Currency string
}

type PhotoInput struct {
	ObjectKey   string
	ContentType string
	SizeBytes   int
}

type VariantInput struct {
	Size        string
	Colour      string
	PriceAmount int
	CostAmount  int
	Currency    string
	Quantity    int
}

type CreateProductInput struct {
	Account        AccountContext
	Name           string
	Category       string
	Photo          PhotoInput
	Variants       []VariantInput
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type UpdateProductInput struct {
	Account        AccountContext
	ProductID      string
	Name           string
	Category       string
	Photo          PhotoInput
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
}

type UpdateVariantInput struct {
	Account        AccountContext
	VariantID      string
	Variant        VariantInput
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
}

type DeleteInventoryInput struct {
	Account        AccountContext
	EntityID       string
	IdempotencyKey string
	RequestID      string
	DeletedAt      time.Time
}

type ListInventoryInput struct {
	Account AccountContext
}

type CreateProductCommand struct {
	Account        AccountContext
	Name           string
	NormalizedName string
	Category       inventorydomain.Category
	Photo          inventorydomain.PhotoMetadata
	Variants       []CreateVariantCommand
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type CreateVariantCommand struct {
	Size             inventorydomain.Size
	Colour           string
	NormalizedColour string
	Price            inventorydomain.Money
	Cost             inventorydomain.Money
	Quantity         int
}

type UpdateProductCommand struct {
	Account        AccountContext
	ProductID      string
	Name           string
	NormalizedName string
	Category       inventorydomain.Category
	Photo          inventorydomain.PhotoMetadata
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
}

type UpdateVariantCommand struct {
	Account          AccountContext
	VariantID        string
	Size             inventorydomain.Size
	Colour           string
	NormalizedColour string
	Price            inventorydomain.Money
	Cost             inventorydomain.Money
	Quantity         int
	IdempotencyKey   string
	RequestID        string
	UpdatedAt        time.Time
}

type SoftDeleteProductCommand struct {
	Account        AccountContext
	ProductID      string
	IdempotencyKey string
	RequestID      string
	DeletedAt      time.Time
}

type SoftDeleteVariantCommand struct {
	Account        AccountContext
	VariantID      string
	IdempotencyKey string
	RequestID      string
	DeletedAt      time.Time
}

type ListInventoryQuery struct {
	Account AccountContext
}

type Product struct {
	ID             string
	BandID         string
	Name           string
	NormalizedName string
	Category       inventorydomain.Category
	Photo          inventorydomain.PhotoMetadata
	Variants       []Variant
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Variant struct {
	ID               string
	ProductID        string
	Size             inventorydomain.Size
	Colour           string
	NormalizedColour string
	Price            inventorydomain.Money
	Cost             inventorydomain.Money
	Quantity         int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func CreateProduct(ctx context.Context, repository Repository, input CreateProductInput) (Product, error) {
	command, err := validateCreateProductInput(input)
	if err != nil {
		return Product{}, err
	}

	product, err := repository.CreateProduct(ctx, command)
	if err != nil {
		return Product{}, fmt.Errorf("create inventory product band_id=%q name=%q category=%q: %w", command.Account.BandID, command.Name, command.Category, err)
	}

	return product, nil
}

func ListInventory(ctx context.Context, repository Repository, input ListInventoryInput) ([]Product, error) {
	query, err := validateListInventoryInput(input)
	if err != nil {
		return nil, err
	}

	products, err := repository.ListInventory(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list inventory band_id=%q: %w", query.Account.BandID, err)
	}

	return products, nil
}

func UpdateProduct(ctx context.Context, repository Repository, input UpdateProductInput) (Product, error) {
	command, err := validateUpdateProductInput(input)
	if err != nil {
		return Product{}, err
	}

	product, err := repository.UpdateProduct(ctx, command)
	if err != nil {
		return Product{}, fmt.Errorf("update inventory product band_id=%q product_id=%q: %w", command.Account.BandID, command.ProductID, err)
	}

	return product, nil
}

func UpdateVariant(ctx context.Context, repository Repository, input UpdateVariantInput) (Variant, error) {
	command, err := validateUpdateVariantInput(input)
	if err != nil {
		return Variant{}, err
	}

	variant, err := repository.UpdateVariant(ctx, command)
	if err != nil {
		return Variant{}, fmt.Errorf("update inventory variant band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
	}

	return variant, nil
}

func SoftDeleteProduct(ctx context.Context, repository Repository, input DeleteInventoryInput) error {
	command, err := validateSoftDeleteProductInput(input)
	if err != nil {
		return err
	}

	if err := repository.SoftDeleteProduct(ctx, command); err != nil {
		return fmt.Errorf("soft delete inventory product band_id=%q product_id=%q: %w", command.Account.BandID, command.ProductID, err)
	}

	return nil
}

func SoftDeleteVariant(ctx context.Context, repository Repository, input DeleteInventoryInput) error {
	command, err := validateSoftDeleteVariantInput(input)
	if err != nil {
		return err
	}

	if err := repository.SoftDeleteVariant(ctx, command); err != nil {
		return fmt.Errorf("soft delete inventory variant band_id=%q variant_id=%q: %w", command.Account.BandID, command.VariantID, err)
	}

	return nil
}

func validateCreateProductInput(input CreateProductInput) (CreateProductCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return CreateProductCommand{}, err
	}

	name, category, normalizedName, photo, err := validateProductFields(input.Name, input.Category, input.Photo)
	if err != nil {
		return CreateProductCommand{}, err
	}

	if len(input.Variants) == 0 {
		return CreateProductCommand{}, fmt.Errorf("at least one inventory variant is required")
	}

	variants, err := validateCreateVariantInputs(input.Variants)
	if err != nil {
		return CreateProductCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.CreatedAt)
	if err != nil {
		return CreateProductCommand{}, err
	}

	return CreateProductCommand{
		Account:        input.Account,
		Name:           name,
		NormalizedName: normalizedName,
		Category:       category,
		Photo:          photo,
		Variants:       variants,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      input.CreatedAt.UTC(),
	}, nil
}

func validateListInventoryInput(input ListInventoryInput) (ListInventoryQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return ListInventoryQuery{}, err
	}

	return ListInventoryQuery{Account: input.Account}, nil
}

func validateUpdateProductInput(input UpdateProductInput) (UpdateProductCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return UpdateProductCommand{}, err
	}

	productID, err := validateID("product id", input.ProductID)
	if err != nil {
		return UpdateProductCommand{}, err
	}

	name, category, normalizedName, photo, err := validateProductFields(input.Name, input.Category, input.Photo)
	if err != nil {
		return UpdateProductCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.UpdatedAt)
	if err != nil {
		return UpdateProductCommand{}, err
	}

	return UpdateProductCommand{
		Account:        input.Account,
		ProductID:      productID,
		Name:           name,
		NormalizedName: normalizedName,
		Category:       category,
		Photo:          photo,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		UpdatedAt:      input.UpdatedAt.UTC(),
	}, nil
}

func validateUpdateVariantInput(input UpdateVariantInput) (UpdateVariantCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return UpdateVariantCommand{}, err
	}

	variantID, err := validateID("variant id", input.VariantID)
	if err != nil {
		return UpdateVariantCommand{}, err
	}

	variant, err := validateVariantInput(input.Variant)
	if err != nil {
		return UpdateVariantCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.UpdatedAt)
	if err != nil {
		return UpdateVariantCommand{}, err
	}

	return UpdateVariantCommand{
		Account:          input.Account,
		VariantID:        variantID,
		Size:             variant.Size,
		Colour:           variant.Colour,
		NormalizedColour: variant.NormalizedColour,
		Price:            variant.Price,
		Cost:             variant.Cost,
		Quantity:         variant.Quantity,
		IdempotencyKey:   idempotencyKey,
		RequestID:        requestID,
		UpdatedAt:        input.UpdatedAt.UTC(),
	}, nil
}

func validateSoftDeleteProductInput(input DeleteInventoryInput) (SoftDeleteProductCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return SoftDeleteProductCommand{}, err
	}

	productID, err := validateID("product id", input.EntityID)
	if err != nil {
		return SoftDeleteProductCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.DeletedAt)
	if err != nil {
		return SoftDeleteProductCommand{}, err
	}

	return SoftDeleteProductCommand{
		Account:        input.Account,
		ProductID:      productID,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		DeletedAt:      input.DeletedAt.UTC(),
	}, nil
}

func validateSoftDeleteVariantInput(input DeleteInventoryInput) (SoftDeleteVariantCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return SoftDeleteVariantCommand{}, err
	}

	variantID, err := validateID("variant id", input.EntityID)
	if err != nil {
		return SoftDeleteVariantCommand{}, err
	}

	idempotencyKey, requestID, err := validateMutationMetadata(input.IdempotencyKey, input.RequestID, input.DeletedAt)
	if err != nil {
		return SoftDeleteVariantCommand{}, err
	}

	return SoftDeleteVariantCommand{
		Account:        input.Account,
		VariantID:      variantID,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		DeletedAt:      input.DeletedAt.UTC(),
	}, nil
}

func validateProductFields(nameInput string, categoryInput string, photoInput PhotoInput) (string, inventorydomain.Category, string, inventorydomain.PhotoMetadata, error) {
	name := strings.TrimSpace(nameInput)
	if name == "" {
		return "", "", "", inventorydomain.PhotoMetadata{}, fmt.Errorf("product name is required")
	}

	category, err := inventorydomain.ParseCategory(categoryInput)
	if err != nil {
		return "", "", "", inventorydomain.PhotoMetadata{}, err
	}

	identity, err := inventorydomain.ProductIdentityFor(category, name)
	if err != nil {
		return "", "", "", inventorydomain.PhotoMetadata{}, err
	}

	photo := inventorydomain.PhotoMetadata{
		ObjectKey:   strings.TrimSpace(photoInput.ObjectKey),
		ContentType: strings.TrimSpace(photoInput.ContentType),
		SizeBytes:   photoInput.SizeBytes,
	}
	if err := inventorydomain.ValidatePhotoMetadata(photo); err != nil {
		return "", "", "", inventorydomain.PhotoMetadata{}, err
	}

	return name, category, identity.NormalizedName, photo, nil
}

func validateCreateVariantInputs(inputs []VariantInput) ([]CreateVariantCommand, error) {
	variants := make([]CreateVariantCommand, 0, len(inputs))
	seenIdentities := make(map[inventorydomain.VariantIdentity]bool, len(inputs))
	for index, input := range inputs {
		variant, err := validateVariantInput(input)
		if err != nil {
			return nil, fmt.Errorf("variant at index %d: %w", index, err)
		}

		identity := inventorydomain.VariantIdentity{
			Size:             variant.Size,
			NormalizedColour: variant.NormalizedColour,
		}
		if seenIdentities[identity] {
			return nil, fmt.Errorf("duplicate variant size %q and colour %q", identity.Size, identity.NormalizedColour)
		}
		seenIdentities[identity] = true

		variants = append(variants, variant)
	}

	return variants, nil
}

func validateVariantInput(input VariantInput) (CreateVariantCommand, error) {
	size, err := inventorydomain.ParseSize(input.Size)
	if err != nil {
		return CreateVariantCommand{}, err
	}

	colour := strings.TrimSpace(input.Colour)
	identity, err := inventorydomain.VariantIdentityFor(size, colour)
	if err != nil {
		return CreateVariantCommand{}, err
	}

	price := inventorydomain.Money{Amount: input.PriceAmount, Currency: strings.TrimSpace(input.Currency)}
	if err := inventorydomain.ValidateMoney("price", price); err != nil {
		return CreateVariantCommand{}, err
	}

	cost := inventorydomain.Money{Amount: input.CostAmount, Currency: strings.TrimSpace(input.Currency)}
	if err := inventorydomain.ValidateMoney("cost", cost); err != nil {
		return CreateVariantCommand{}, err
	}

	if err := inventorydomain.ValidateQuantity(input.Quantity); err != nil {
		return CreateVariantCommand{}, err
	}

	return CreateVariantCommand{
		Size:             size,
		Colour:           colour,
		NormalizedColour: identity.NormalizedColour,
		Price:            price,
		Cost:             cost,
		Quantity:         input.Quantity,
	}, nil
}

func validateReadAccount(account AccountContext) error {
	if strings.TrimSpace(account.UserID) == "" {
		return fmt.Errorf("user id is required")
	}

	if strings.TrimSpace(account.BandID) == "" {
		return fmt.Errorf("band id is required")
	}

	if !permissions.CanReadInAlpha(account.Role) {
		return fmt.Errorf("alpha read access denied for role %q", account.Role)
	}

	return nil
}

func validateWriteAccount(account AccountContext) error {
	if err := validateReadAccount(account); err != nil {
		return err
	}

	if err := permissions.RequireAlphaWrite(account.Role); err != nil {
		return err
	}

	return nil
}

func validateMutationMetadata(idempotencyKeyInput string, requestIDInput string, timestamp time.Time) (string, string, error) {
	idempotencyKey := strings.TrimSpace(idempotencyKeyInput)
	if idempotencyKey == "" {
		return "", "", fmt.Errorf("idempotency key is required")
	}

	requestID := strings.TrimSpace(requestIDInput)
	if requestID == "" {
		return "", "", fmt.Errorf("request id is required")
	}

	if timestamp.IsZero() {
		return "", "", fmt.Errorf("mutation timestamp is required")
	}

	return idempotencyKey, requestID, nil
}

func validateID(label string, value string) (string, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return "", fmt.Errorf("%s is required", label)
	}

	if _, err := uuid.Parse(trimmedValue); err != nil {
		return "", fmt.Errorf("%s must be a valid UUID", label)
	}

	return trimmedValue, nil
}

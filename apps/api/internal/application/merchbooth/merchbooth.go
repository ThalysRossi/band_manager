package merchbooth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	"github.com/thalys/band-manager/apps/api/internal/domain/permissions"
)

var (
	ErrEmptyCart           = errors.New("cart is empty")
	ErrDuplicateCartItem   = errors.New("duplicate cart item")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrBoothItemNotFound   = errors.New("booth item not found")
	ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")
)

type Repository interface {
	ListBoothItems(ctx context.Context, query ListBoothItemsQuery) ([]BoothItem, error)
	CreateCashCheckout(ctx context.Context, command CreateCashCheckoutCommand) (Sale, error)
}

type AccountContext struct {
	UserID string
	BandID string
	Role   permissions.Role
}

type CartItemInput struct {
	VariantID string
	Quantity  int
}

type ListBoothItemsInput struct {
	Account AccountContext
}

type CreateCashCheckoutInput struct {
	Account        AccountContext
	Items          []CartItemInput
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type ListBoothItemsQuery struct {
	Account AccountContext
}

type CreateCashCheckoutCommand struct {
	Account        AccountContext
	Items          []CartItem
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type CartItem struct {
	VariantID string
	Quantity  int
}

type BoothItem struct {
	ProductID   string
	VariantID   string
	ProductName string
	Category    inventorydomain.Category
	Size        inventorydomain.Size
	Colour      string
	Price       inventorydomain.Money
	Cost        inventorydomain.Money
	Quantity    int
	Photo       inventorydomain.PhotoMetadata
	SoldOut     bool
}

type Sale struct {
	ID             string
	BandID         string
	Status         SaleStatus
	Total          inventorydomain.Money
	ExpectedProfit inventorydomain.Money
	Items          []SaleItem
	Payment        Payment
	Transactions   []Transaction
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SaleItem struct {
	ID             string
	SaleID         string
	ProductID      string
	VariantID      string
	ProductName    string
	Category       inventorydomain.Category
	Size           inventorydomain.Size
	Colour         string
	Quantity       int
	UnitPrice      inventorydomain.Money
	UnitCost       inventorydomain.Money
	LineTotal      inventorydomain.Money
	ExpectedProfit inventorydomain.Money
	CreatedAt      time.Time
}

type Payment struct {
	ID        string
	SaleID    string
	Method    PaymentMethod
	Status    PaymentStatus
	Amount    inventorydomain.Money
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Transaction struct {
	ID         string
	SaleID     string
	SaleItemID string
	Amount     inventorydomain.Money
	CreatedAt  time.Time
}

type SaleStatus string

const (
	SaleStatusFinalized SaleStatus = "finalized"
)

type PaymentMethod string

const (
	PaymentMethodCash PaymentMethod = "cash"
)

type PaymentStatus string

const (
	PaymentStatusConfirmed PaymentStatus = "confirmed"
)

func ListBoothItems(ctx context.Context, repository Repository, input ListBoothItemsInput) ([]BoothItem, error) {
	query, err := validateListBoothItemsInput(input)
	if err != nil {
		return nil, err
	}

	items, err := repository.ListBoothItems(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list merch booth items band_id=%q: %w", query.Account.BandID, err)
	}

	return items, nil
}

func CreateCashCheckout(ctx context.Context, repository Repository, input CreateCashCheckoutInput) (Sale, error) {
	command, err := validateCreateCashCheckoutInput(input)
	if err != nil {
		return Sale{}, err
	}

	sale, err := repository.CreateCashCheckout(ctx, command)
	if err != nil {
		return Sale{}, fmt.Errorf("create cash checkout band_id=%q: %w", command.Account.BandID, err)
	}

	return sale, nil
}

func validateListBoothItemsInput(input ListBoothItemsInput) (ListBoothItemsQuery, error) {
	if err := validateReadAccount(input.Account); err != nil {
		return ListBoothItemsQuery{}, err
	}

	return ListBoothItemsQuery{Account: input.Account}, nil
}

func validateCreateCashCheckoutInput(input CreateCashCheckoutInput) (CreateCashCheckoutCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return CreateCashCheckoutCommand{}, err
	}

	if len(input.Items) == 0 {
		return CreateCashCheckoutCommand{}, ErrEmptyCart
	}

	items := make([]CartItem, 0, len(input.Items))
	seenVariantIDs := make(map[string]bool, len(input.Items))
	for index, inputItem := range input.Items {
		variantID, err := validateID("variant id", inputItem.VariantID)
		if err != nil {
			return CreateCashCheckoutCommand{}, fmt.Errorf("cart item at index %d: %w", index, err)
		}

		if seenVariantIDs[variantID] {
			return CreateCashCheckoutCommand{}, fmt.Errorf("%w: variant_id=%q", ErrDuplicateCartItem, variantID)
		}
		seenVariantIDs[variantID] = true

		if inputItem.Quantity <= 0 {
			return CreateCashCheckoutCommand{}, fmt.Errorf("cart item at index %d: quantity must be greater than zero", index)
		}

		items = append(items, CartItem{
			VariantID: variantID,
			Quantity:  inputItem.Quantity,
		})
	}

	sort.Slice(items, func(leftIndex int, rightIndex int) bool {
		return items[leftIndex].VariantID < items[rightIndex].VariantID
	})

	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		return CreateCashCheckoutCommand{}, fmt.Errorf("idempotency key is required")
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		return CreateCashCheckoutCommand{}, fmt.Errorf("request id is required")
	}

	if input.CreatedAt.IsZero() {
		return CreateCashCheckoutCommand{}, fmt.Errorf("created at timestamp is required")
	}

	return CreateCashCheckoutCommand{
		Account:        input.Account,
		Items:          items,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      input.CreatedAt.UTC(),
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

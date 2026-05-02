package merchbooth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	ErrPaymentProvider     = errors.New("payment provider error")
)

type Repository interface {
	ListBoothItems(ctx context.Context, query ListBoothItemsQuery) ([]BoothItem, error)
	CreateCashCheckout(ctx context.Context, command CreateCashCheckoutCommand) (Sale, error)
	ReservePixCheckout(ctx context.Context, command CreatePixCheckoutCommand) (Sale, bool, error)
	CompletePixCheckoutPayment(ctx context.Context, command CompletePixCheckoutPaymentCommand) (Sale, error)
	FailPixCheckoutPaymentCreation(ctx context.Context, command FailPixCheckoutPaymentCreationCommand) error
}

type PaymentProvider interface {
	CreatePixPayment(ctx context.Context, command CreatePixPaymentCommand) (PixPayment, error)
}

type AccountContext struct {
	UserID string
	BandID string
	Email  string
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

type CreatePixCheckoutInput struct {
	Account        AccountContext
	Items          []CartItemInput
	PayerEmail     string
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

type CreatePixCheckoutCommand struct {
	Account           AccountContext
	SaleID            string
	PaymentID         string
	ExternalReference string
	Items             []CartItem
	PayerEmail        string
	IdempotencyKey    string
	RequestID         string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

type CompletePixCheckoutPaymentCommand struct {
	Account        AccountContext
	SaleID         string
	PaymentID      string
	RequestID      string
	RequestHash    string
	ProviderResult PixPayment
	IdempotencyKey string
	UpdatedAt      time.Time
}

type FailPixCheckoutPaymentCreationCommand struct {
	Account        AccountContext
	SaleID         string
	PaymentID      string
	RequestID      string
	IdempotencyKey string
	UpdatedAt      time.Time
}

type CreatePixPaymentCommand struct {
	SaleID            string
	PaymentID         string
	ExternalReference string
	Amount            inventorydomain.Money
	PayerEmail        string
	IdempotencyKey    string
	ExpiresAt         time.Time
}

type PixPayment struct {
	Provider             string
	ProviderOrderID      string
	ProviderPaymentID    string
	ProviderReferenceID  string
	ExternalReference    string
	ProviderStatus       string
	ProviderStatusDetail string
	LocalStatus          PaymentStatus
	Amount               inventorydomain.Money
	ExpiresAt            time.Time
	QRCode               string
	QRCodeBase64         string
	TicketURL            string
	RawProviderResponse  []byte
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
	ID                   string
	SaleID               string
	Method               PaymentMethod
	Status               PaymentStatus
	Amount               inventorydomain.Money
	Provider             string
	ProviderOrderID      string
	ProviderPaymentID    string
	ProviderReferenceID  string
	ExternalReference    string
	ProviderStatus       string
	ProviderStatusDetail string
	ExpiresAt            time.Time
	PixQRCode            string
	PixQRCodeBase64      string
	PixTicketURL         string
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
	SaleStatusFinalized      SaleStatus = "finalized"
	SaleStatusPendingPayment SaleStatus = "pending_payment"
	SaleStatusCanceled       SaleStatus = "canceled"
)

type PaymentMethod string

const (
	PaymentMethodCash PaymentMethod = "cash"
	PaymentMethodPix  PaymentMethod = "pix"
)

type PaymentStatus string

const (
	PaymentStatusConfirmed       PaymentStatus = "confirmed"
	PaymentStatusProviderPending PaymentStatus = "provider_pending"
	PaymentStatusActionRequired  PaymentStatus = "action_required"
	PaymentStatusProcessing      PaymentStatus = "processing"
	PaymentStatusFailed          PaymentStatus = "failed"
	PaymentStatusCanceled        PaymentStatus = "canceled"
	PaymentStatusExpired         PaymentStatus = "expired"
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

func CreatePixCheckout(ctx context.Context, repository Repository, paymentProvider PaymentProvider, input CreatePixCheckoutInput) (Sale, error) {
	command, requestHash, err := validateCreatePixCheckoutInput(input)
	if err != nil {
		return Sale{}, err
	}

	reservedSale, found, err := repository.ReservePixCheckout(ctx, command)
	if err != nil {
		return Sale{}, fmt.Errorf("reserve pix checkout band_id=%q: %w", command.Account.BandID, err)
	}
	if found {
		return reservedSale, nil
	}

	providerResult, err := paymentProvider.CreatePixPayment(ctx, CreatePixPaymentCommand{
		SaleID:            command.SaleID,
		PaymentID:         command.PaymentID,
		ExternalReference: command.ExternalReference,
		Amount:            reservedSale.Total,
		PayerEmail:        command.PayerEmail,
		IdempotencyKey:    command.IdempotencyKey,
		ExpiresAt:         command.ExpiresAt,
	})
	if err != nil {
		releaseErr := repository.FailPixCheckoutPaymentCreation(ctx, FailPixCheckoutPaymentCreationCommand{
			Account:        command.Account,
			SaleID:         command.SaleID,
			PaymentID:      command.PaymentID,
			RequestID:      command.RequestID,
			IdempotencyKey: command.IdempotencyKey,
			UpdatedAt:      command.CreatedAt,
		})
		if releaseErr != nil {
			return Sale{}, fmt.Errorf("create pix payment failed and reservation release failed band_id=%q sale_id=%q: provider_error=%w release_error=%v", command.Account.BandID, command.SaleID, err, releaseErr)
		}

		return Sale{}, fmt.Errorf("%w: create pix payment band_id=%q sale_id=%q: %v", ErrPaymentProvider, command.Account.BandID, command.SaleID, err)
	}

	sale, err := repository.CompletePixCheckoutPayment(ctx, CompletePixCheckoutPaymentCommand{
		Account:        command.Account,
		SaleID:         command.SaleID,
		PaymentID:      command.PaymentID,
		RequestID:      command.RequestID,
		RequestHash:    requestHash,
		ProviderResult: providerResult,
		IdempotencyKey: command.IdempotencyKey,
		UpdatedAt:      command.CreatedAt,
	})
	if err != nil {
		return Sale{}, fmt.Errorf("complete pix checkout payment band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
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

func validateCreatePixCheckoutInput(input CreatePixCheckoutInput) (CreatePixCheckoutCommand, string, error) {
	cashLikeInput := CreateCashCheckoutInput{
		Account:        input.Account,
		Items:          input.Items,
		IdempotencyKey: input.IdempotencyKey,
		RequestID:      input.RequestID,
		CreatedAt:      input.CreatedAt,
	}
	cashCommand, err := validateCreateCashCheckoutInput(cashLikeInput)
	if err != nil {
		return CreatePixCheckoutCommand{}, "", err
	}

	payerEmail := strings.TrimSpace(input.PayerEmail)
	if payerEmail == "" {
		return CreatePixCheckoutCommand{}, "", fmt.Errorf("payer email is required")
	}
	if !strings.Contains(payerEmail, "@") {
		return CreatePixCheckoutCommand{}, "", fmt.Errorf("payer email %q must contain @", payerEmail)
	}

	saleID := uuid.NewString()
	paymentID := uuid.NewString()
	command := CreatePixCheckoutCommand{
		Account:           cashCommand.Account,
		SaleID:            saleID,
		PaymentID:         paymentID,
		ExternalReference: "sale_" + saleID,
		Items:             cashCommand.Items,
		PayerEmail:        payerEmail,
		IdempotencyKey:    cashCommand.IdempotencyKey,
		RequestID:         cashCommand.RequestID,
		CreatedAt:         cashCommand.CreatedAt,
		ExpiresAt:         cashCommand.CreatedAt.Add(30 * time.Minute),
	}

	requestHash, err := HashPixCheckoutRequest(command)
	if err != nil {
		return CreatePixCheckoutCommand{}, "", err
	}

	return command, requestHash, nil
}

func HashPixCheckoutRequest(command CreatePixCheckoutCommand) (string, error) {
	body, err := json.Marshal(struct {
		BandID     string        `json:"bandId"`
		Items      []CartItem    `json:"items"`
		Method     PaymentMethod `json:"method"`
		PayerEmail string        `json:"payerEmail"`
	}{
		BandID:     command.Account.BandID,
		Items:      command.Items,
		Method:     PaymentMethodPix,
		PayerEmail: command.PayerEmail,
	})
	if err != nil {
		return "", fmt.Errorf("marshal pix checkout request hash body: %w", err)
	}

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
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

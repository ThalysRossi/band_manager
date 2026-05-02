package merchbooth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	ErrWebhookSignature    = errors.New("webhook signature is invalid")
	ErrPaymentNotFound     = errors.New("payment not found")
)

type Repository interface {
	ListBoothItems(ctx context.Context, query ListBoothItemsQuery) ([]BoothItem, error)
	CreateCashCheckout(ctx context.Context, command CreateCashCheckoutCommand) (Sale, error)
	ReservePixCheckout(ctx context.Context, command CreatePixCheckoutCommand) (Sale, bool, error)
	ReserveCardCheckout(ctx context.Context, command CreateCardCheckoutCommand) (Sale, bool, error)
	CompletePixCheckoutPayment(ctx context.Context, command CompletePixCheckoutPaymentCommand) (Sale, error)
	CompleteCardCheckoutPayment(ctx context.Context, command CompleteCardCheckoutPaymentCommand) (Sale, error)
	FailPixCheckoutPaymentCreation(ctx context.Context, command FailPixCheckoutPaymentCreationCommand) error
	FailCardCheckoutPaymentCreation(ctx context.Context, command FailCardCheckoutPaymentCreationCommand) error
	GetPixPaymentProviderOrderID(ctx context.Context, query GetPixPaymentProviderOrderIDQuery) (string, error)
	ApplyPixPaymentStatus(ctx context.Context, command ApplyPixPaymentStatusCommand) (Sale, error)
	RecordPaymentEvent(ctx context.Context, command PaymentEventCommand) error
}

type PaymentProvider interface {
	CreatePixPayment(ctx context.Context, command CreatePixPaymentCommand) (PixPayment, error)
	CreateCardPayment(ctx context.Context, command CreateCardPaymentCommand) (PixPayment, error)
	GetPaymentStatus(ctx context.Context, command GetPaymentStatusCommand) (PixPayment, error)
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

type CreateCardCheckoutInput struct {
	Account        AccountContext
	Items          []CartItemInput
	CardType       CardPaymentType
	TerminalID     string
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type VerifyPixPaymentInput struct {
	Account        AccountContext
	PaymentID      string
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
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

type CreateCardCheckoutCommand struct {
	Account           AccountContext
	SaleID            string
	PaymentID         string
	ExternalReference string
	Items             []CartItem
	CardType          CardPaymentType
	TerminalID        string
	Installments      int
	IdempotencyKey    string
	RequestID         string
	CreatedAt         time.Time
	ExpiresAt         time.Time
}

type VerifyPixPaymentCommand struct {
	Account        AccountContext
	PaymentID      string
	IdempotencyKey string
	RequestID      string
	UpdatedAt      time.Time
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

type CompleteCardCheckoutPaymentCommand struct {
	Account        AccountContext
	SaleID         string
	PaymentID      string
	RequestID      string
	RequestHash    string
	ProviderResult PixPayment
	CardType       CardPaymentType
	TerminalID     string
	Installments   int
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

type FailCardCheckoutPaymentCreationCommand struct {
	Account        AccountContext
	SaleID         string
	PaymentID      string
	RequestID      string
	IdempotencyKey string
	UpdatedAt      time.Time
}

type GetPixPaymentProviderOrderIDQuery struct {
	Account   AccountContext
	PaymentID string
}

type ApplyPixPaymentStatusCommand struct {
	ProviderResult PixPayment
	RequestID      string
	IdempotencyKey string
	UpdatedAt      time.Time
}

type PaymentEventCommand struct {
	Provider           string
	ProviderOrderID    string
	BandID             string
	SaleID             string
	PaymentID          string
	WebhookRequestID   string
	SignatureTimestamp time.Time
	SignatureVerified  bool
	RawQuery           string
	RawBody            []byte
	ProcessingStatus   PaymentEventProcessingStatus
	ProcessingError    string
	ReceivedAt         time.Time
	ProcessedAt        time.Time
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

type CreateCardPaymentCommand struct {
	SaleID            string
	PaymentID         string
	ExternalReference string
	Amount            inventorydomain.Money
	TerminalID        string
	CardType          CardPaymentType
	Installments      int
	IdempotencyKey    string
	ExpiresAt         time.Time
}

type GetPaymentStatusCommand struct {
	ProviderOrderID string
}

type MercadoPagoOrderWebhookInput struct {
	DataID          string
	Type            string
	SignatureHeader string
	RequestID       string
	WebhookSecret   string
	RawQuery        string
	RawBody         []byte
	ReceivedAt      time.Time
	Now             time.Time
}

type VerifiedMercadoPagoWebhook struct {
	ProviderOrderID    string
	RequestID          string
	SignatureTimestamp time.Time
	RawQuery           string
	RawBody            []byte
	ReceivedAt         time.Time
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
	PointTerminalID      string
	CardPaymentType      CardPaymentType
	CardInstallments     int
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
	PaymentMethodCard PaymentMethod = "card"
)

type CardPaymentType string

const (
	CardPaymentTypeCredit CardPaymentType = "credit_card"
	CardPaymentTypeDebit  CardPaymentType = "debit_card"
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

type PaymentEventProcessingStatus string

const (
	PaymentEventProcessingStatusProcessed PaymentEventProcessingStatus = "processed"
	PaymentEventProcessingStatusRejected  PaymentEventProcessingStatus = "rejected"
	PaymentEventProcessingStatusFailed    PaymentEventProcessingStatus = "failed"
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

func HandleMercadoPagoOrderWebhook(ctx context.Context, repository Repository, paymentProvider PaymentProvider, input MercadoPagoOrderWebhookInput) (Sale, error) {
	verifiedWebhook, err := VerifyMercadoPagoOrderWebhookSignature(input)
	if err != nil {
		recordErr := repository.RecordPaymentEvent(ctx, PaymentEventCommand{
			Provider:          "mercadopago",
			ProviderOrderID:   strings.TrimSpace(input.DataID),
			WebhookRequestID:  strings.TrimSpace(input.RequestID),
			SignatureVerified: false,
			RawQuery:          input.RawQuery,
			RawBody:           input.RawBody,
			ProcessingStatus:  PaymentEventProcessingStatusRejected,
			ProcessingError:   err.Error(),
			ReceivedAt:        input.ReceivedAt.UTC(),
			ProcessedAt:       input.Now.UTC(),
		})
		if recordErr != nil {
			return Sale{}, fmt.Errorf("%w: %v; record payment event failed: %v", ErrWebhookSignature, err, recordErr)
		}

		return Sale{}, fmt.Errorf("%w: %v", ErrWebhookSignature, err)
	}

	providerResult, err := paymentProvider.GetPaymentStatus(ctx, GetPaymentStatusCommand{ProviderOrderID: verifiedWebhook.ProviderOrderID})
	if err != nil {
		recordErr := repository.RecordPaymentEvent(ctx, failedPaymentEventCommand(verifiedWebhook, err, input.Now.UTC()))
		if recordErr != nil {
			return Sale{}, fmt.Errorf("%w: get payment status provider_order_id=%q: %v; record payment event failed: %v", ErrPaymentProvider, verifiedWebhook.ProviderOrderID, err, recordErr)
		}

		return Sale{}, fmt.Errorf("%w: get payment status provider_order_id=%q: %v", ErrPaymentProvider, verifiedWebhook.ProviderOrderID, err)
	}

	sale, err := repository.ApplyPixPaymentStatus(ctx, ApplyPixPaymentStatusCommand{
		ProviderResult: providerResult,
		RequestID:      verifiedWebhook.RequestID,
		IdempotencyKey: verifiedWebhook.RequestID,
		UpdatedAt:      input.Now.UTC(),
	})
	if err != nil {
		recordErr := repository.RecordPaymentEvent(ctx, failedPaymentEventCommand(verifiedWebhook, err, input.Now.UTC()))
		if recordErr != nil {
			return Sale{}, fmt.Errorf("apply pix payment status provider_order_id=%q: %w; record payment event failed: %v", verifiedWebhook.ProviderOrderID, err, recordErr)
		}

		return Sale{}, fmt.Errorf("apply pix payment status provider_order_id=%q: %w", verifiedWebhook.ProviderOrderID, err)
	}

	if err := repository.RecordPaymentEvent(ctx, PaymentEventCommand{
		Provider:           "mercadopago",
		ProviderOrderID:    verifiedWebhook.ProviderOrderID,
		BandID:             sale.BandID,
		SaleID:             sale.ID,
		PaymentID:          sale.Payment.ID,
		WebhookRequestID:   verifiedWebhook.RequestID,
		SignatureTimestamp: verifiedWebhook.SignatureTimestamp,
		SignatureVerified:  true,
		RawQuery:           verifiedWebhook.RawQuery,
		RawBody:            verifiedWebhook.RawBody,
		ProcessingStatus:   PaymentEventProcessingStatusProcessed,
		ReceivedAt:         verifiedWebhook.ReceivedAt,
		ProcessedAt:        input.Now.UTC(),
	}); err != nil {
		return Sale{}, fmt.Errorf("record processed payment event provider_order_id=%q: %w", verifiedWebhook.ProviderOrderID, err)
	}

	return sale, nil
}

func VerifyPixPayment(ctx context.Context, repository Repository, paymentProvider PaymentProvider, input VerifyPixPaymentInput) (Sale, error) {
	command, err := validateVerifyPixPaymentInput(input)
	if err != nil {
		return Sale{}, err
	}

	providerOrderID, err := repository.GetPixPaymentProviderOrderID(ctx, GetPixPaymentProviderOrderIDQuery{
		Account:   command.Account,
		PaymentID: command.PaymentID,
	})
	if err != nil {
		return Sale{}, fmt.Errorf("get pix payment provider order id band_id=%q payment_id=%q: %w", command.Account.BandID, command.PaymentID, err)
	}

	providerResult, err := paymentProvider.GetPaymentStatus(ctx, GetPaymentStatusCommand{ProviderOrderID: providerOrderID})
	if err != nil {
		return Sale{}, fmt.Errorf("%w: get payment status provider_order_id=%q: %v", ErrPaymentProvider, providerOrderID, err)
	}

	sale, err := repository.ApplyPixPaymentStatus(ctx, ApplyPixPaymentStatusCommand{
		ProviderResult: providerResult,
		RequestID:      command.RequestID,
		IdempotencyKey: command.IdempotencyKey,
		UpdatedAt:      command.UpdatedAt,
	})
	if err != nil {
		return Sale{}, fmt.Errorf("apply pix payment status band_id=%q payment_id=%q provider_order_id=%q: %w", command.Account.BandID, command.PaymentID, providerOrderID, err)
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

func CreateCardCheckout(ctx context.Context, repository Repository, paymentProvider PaymentProvider, input CreateCardCheckoutInput) (Sale, error) {
	command, requestHash, err := validateCreateCardCheckoutInput(input)
	if err != nil {
		return Sale{}, err
	}

	reservedSale, found, err := repository.ReserveCardCheckout(ctx, command)
	if err != nil {
		return Sale{}, fmt.Errorf("reserve card checkout band_id=%q: %w", command.Account.BandID, err)
	}
	if found {
		return reservedSale, nil
	}

	providerResult, err := paymentProvider.CreateCardPayment(ctx, CreateCardPaymentCommand{
		SaleID:            command.SaleID,
		PaymentID:         command.PaymentID,
		ExternalReference: command.ExternalReference,
		Amount:            reservedSale.Total,
		TerminalID:        command.TerminalID,
		CardType:          command.CardType,
		Installments:      command.Installments,
		IdempotencyKey:    command.IdempotencyKey,
		ExpiresAt:         command.ExpiresAt,
	})
	if err != nil {
		releaseErr := repository.FailCardCheckoutPaymentCreation(ctx, FailCardCheckoutPaymentCreationCommand{
			Account:        command.Account,
			SaleID:         command.SaleID,
			PaymentID:      command.PaymentID,
			RequestID:      command.RequestID,
			IdempotencyKey: command.IdempotencyKey,
			UpdatedAt:      command.CreatedAt,
		})
		if releaseErr != nil {
			return Sale{}, fmt.Errorf("create card payment failed and reservation release failed band_id=%q sale_id=%q: provider_error=%w release_error=%v", command.Account.BandID, command.SaleID, err, releaseErr)
		}

		return Sale{}, fmt.Errorf("%w: create card payment band_id=%q sale_id=%q: %v", ErrPaymentProvider, command.Account.BandID, command.SaleID, err)
	}

	sale, err := repository.CompleteCardCheckoutPayment(ctx, CompleteCardCheckoutPaymentCommand{
		Account:        command.Account,
		SaleID:         command.SaleID,
		PaymentID:      command.PaymentID,
		RequestID:      command.RequestID,
		RequestHash:    requestHash,
		ProviderResult: providerResult,
		CardType:       command.CardType,
		TerminalID:     command.TerminalID,
		Installments:   command.Installments,
		IdempotencyKey: command.IdempotencyKey,
		UpdatedAt:      command.CreatedAt,
	})
	if err != nil {
		return Sale{}, fmt.Errorf("complete card checkout payment band_id=%q sale_id=%q: %w", command.Account.BandID, command.SaleID, err)
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

func validateCreateCardCheckoutInput(input CreateCardCheckoutInput) (CreateCardCheckoutCommand, string, error) {
	cashLikeInput := CreateCashCheckoutInput{
		Account:        input.Account,
		Items:          input.Items,
		IdempotencyKey: input.IdempotencyKey,
		RequestID:      input.RequestID,
		CreatedAt:      input.CreatedAt,
	}
	cashCommand, err := validateCreateCashCheckoutInput(cashLikeInput)
	if err != nil {
		return CreateCardCheckoutCommand{}, "", err
	}

	cardType, err := validateCardPaymentType(input.CardType)
	if err != nil {
		return CreateCardCheckoutCommand{}, "", err
	}

	terminalID := strings.TrimSpace(input.TerminalID)
	if terminalID == "" {
		return CreateCardCheckoutCommand{}, "", fmt.Errorf("point terminal id is required")
	}

	saleID := uuid.NewString()
	paymentID := uuid.NewString()
	command := CreateCardCheckoutCommand{
		Account:           cashCommand.Account,
		SaleID:            saleID,
		PaymentID:         paymentID,
		ExternalReference: "sale_" + saleID,
		Items:             cashCommand.Items,
		CardType:          cardType,
		TerminalID:        terminalID,
		Installments:      1,
		IdempotencyKey:    cashCommand.IdempotencyKey,
		RequestID:         cashCommand.RequestID,
		CreatedAt:         cashCommand.CreatedAt,
		ExpiresAt:         cashCommand.CreatedAt.Add(16 * time.Minute),
	}

	requestHash, err := HashCardCheckoutRequest(command)
	if err != nil {
		return CreateCardCheckoutCommand{}, "", err
	}

	return command, requestHash, nil
}

func validateCardPaymentType(value CardPaymentType) (CardPaymentType, error) {
	switch value {
	case CardPaymentTypeCredit, CardPaymentTypeDebit:
		return value, nil
	default:
		return "", fmt.Errorf("card payment type must be credit_card or debit_card")
	}
}

func validateVerifyPixPaymentInput(input VerifyPixPaymentInput) (VerifyPixPaymentCommand, error) {
	if err := validateWriteAccount(input.Account); err != nil {
		return VerifyPixPaymentCommand{}, err
	}

	paymentID, err := validateID("payment id", input.PaymentID)
	if err != nil {
		return VerifyPixPaymentCommand{}, err
	}

	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		return VerifyPixPaymentCommand{}, fmt.Errorf("idempotency key is required")
	}

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		return VerifyPixPaymentCommand{}, fmt.Errorf("request id is required")
	}

	if input.UpdatedAt.IsZero() {
		return VerifyPixPaymentCommand{}, fmt.Errorf("updated at timestamp is required")
	}

	return VerifyPixPaymentCommand{
		Account:        input.Account,
		PaymentID:      paymentID,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		UpdatedAt:      input.UpdatedAt.UTC(),
	}, nil
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

func HashCardCheckoutRequest(command CreateCardCheckoutCommand) (string, error) {
	body, err := json.Marshal(struct {
		BandID       string          `json:"bandId"`
		Items        []CartItem      `json:"items"`
		Method       PaymentMethod   `json:"method"`
		CardType     CardPaymentType `json:"cardType"`
		TerminalID   string          `json:"terminalId"`
		Installments int             `json:"installments"`
	}{
		BandID:       command.Account.BandID,
		Items:        command.Items,
		Method:       PaymentMethodCard,
		CardType:     command.CardType,
		TerminalID:   command.TerminalID,
		Installments: command.Installments,
	})
	if err != nil {
		return "", fmt.Errorf("marshal card checkout request hash body: %w", err)
	}

	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func VerifyMercadoPagoOrderWebhookSignature(input MercadoPagoOrderWebhookInput) (VerifiedMercadoPagoWebhook, error) {
	if strings.TrimSpace(input.Type) != "order" {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("webhook type must be order")
	}

	providerOrderID := strings.TrimSpace(input.DataID)
	if providerOrderID == "" {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("webhook data.id is required")
	}
	signatureOrderID := normalizeWebhookDataID(providerOrderID)

	requestID := strings.TrimSpace(input.RequestID)
	if requestID == "" {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("webhook x-request-id header is required")
	}

	webhookSecret := strings.TrimSpace(input.WebhookSecret)
	if webhookSecret == "" {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("mercadopago webhook secret is required")
	}

	signatureTimestamp, signatureValue, err := parseMercadoPagoSignatureHeader(input.SignatureHeader)
	if err != nil {
		return VerifiedMercadoPagoWebhook{}, err
	}

	now := input.Now.UTC()
	if now.IsZero() {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("current timestamp is required")
	}

	signatureTime := time.UnixMilli(signatureTimestamp).UTC()
	maxAge := 5 * time.Minute
	if signatureTime.Before(now.Add(-maxAge)) || signatureTime.After(now.Add(maxAge)) {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("webhook signature timestamp is outside tolerance")
	}

	manifest := mercadoPagoSignatureManifest(signatureOrderID, requestID, strconv.FormatInt(signatureTimestamp, 10))
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	if _, err := mac.Write([]byte(manifest)); err != nil {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("compute webhook signature manifest: %w", err)
	}
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(signatureValue)) != 1 {
		return VerifiedMercadoPagoWebhook{}, fmt.Errorf("webhook signature mismatch")
	}

	return VerifiedMercadoPagoWebhook{
		ProviderOrderID:    providerOrderID,
		RequestID:          requestID,
		SignatureTimestamp: signatureTime,
		RawQuery:           input.RawQuery,
		RawBody:            input.RawBody,
		ReceivedAt:         input.ReceivedAt.UTC(),
	}, nil
}

func parseMercadoPagoSignatureHeader(value string) (int64, string, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return 0, "", fmt.Errorf("webhook x-signature header is required")
	}

	parts := strings.Split(trimmedValue, ",")
	values := make(map[string]string, len(parts))
	for _, part := range parts {
		keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(keyValue) != 2 {
			return 0, "", fmt.Errorf("webhook x-signature header contains invalid segment %q", part)
		}
		values[strings.TrimSpace(keyValue[0])] = strings.TrimSpace(keyValue[1])
	}

	rawTimestamp := values["ts"]
	if rawTimestamp == "" {
		return 0, "", fmt.Errorf("webhook x-signature ts value is required")
	}
	signatureTimestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("webhook x-signature ts value %q is invalid: %w", rawTimestamp, err)
	}

	signatureValue := values["v1"]
	if signatureValue == "" {
		return 0, "", fmt.Errorf("webhook x-signature v1 value is required")
	}

	return signatureTimestamp, signatureValue, nil
}

func mercadoPagoSignatureManifest(providerOrderID string, requestID string, signatureTimestamp string) string {
	return "id:" + providerOrderID + ";request-id:" + requestID + ";ts:" + signatureTimestamp + ";"
}

func normalizeWebhookDataID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func failedPaymentEventCommand(webhook VerifiedMercadoPagoWebhook, processingErr error, processedAt time.Time) PaymentEventCommand {
	return PaymentEventCommand{
		Provider:           "mercadopago",
		ProviderOrderID:    webhook.ProviderOrderID,
		WebhookRequestID:   webhook.RequestID,
		SignatureTimestamp: webhook.SignatureTimestamp,
		SignatureVerified:  true,
		RawQuery:           webhook.RawQuery,
		RawBody:            webhook.RawBody,
		ProcessingStatus:   PaymentEventProcessingStatusFailed,
		ProcessingError:    processingErr.Error(),
		ReceivedAt:         webhook.ReceivedAt,
		ProcessedAt:        processedAt.UTC(),
	}
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

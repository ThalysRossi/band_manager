package inventoryhandler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	inventorydomain "github.com/thalys/band-manager/apps/api/internal/domain/inventory"
	authhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware/authcontext"
)

type Handler struct {
	inventoryRepository applicationinventory.Repository
	photoStorage        applicationinventory.PhotoStorage
	logger              *slog.Logger
	now                 func() time.Time
}

type ProductRequest struct {
	Name     string       `json:"name"`
	Category string       `json:"category"`
	Photo    PhotoRequest `json:"photo"`
}

type CreateProductRequest struct {
	Name     string           `json:"name"`
	Category string           `json:"category"`
	Photo    PhotoRequest     `json:"photo"`
	Variants []VariantRequest `json:"variants"`
}

type VariantRequest struct {
	Size     string       `json:"size"`
	Colour   string       `json:"colour"`
	Price    MoneyRequest `json:"price"`
	Cost     MoneyRequest `json:"cost"`
	Quantity *int         `json:"quantity"`
}

type MoneyRequest struct {
	Amount   *int   `json:"amount"`
	Currency string `json:"currency"`
}

type PhotoRequest struct {
	Full    PhotoVariantRequest `json:"full"`
	Display PhotoVariantRequest `json:"display"`
}

type PhotoVariantRequest struct {
	ObjectKey   string `json:"objectKey"`
	ContentType string `json:"contentType"`
	SizeBytes   *int   `json:"sizeBytes"`
	Width       *int   `json:"width"`
	Height      *int   `json:"height"`
}

type PhotoUploadRequest struct {
	Full    PhotoUploadVariantRequest `json:"full"`
	Display PhotoUploadVariantRequest `json:"display"`
}

type PhotoUploadVariantRequest struct {
	ContentType string `json:"contentType"`
	SizeBytes   *int   `json:"sizeBytes"`
	Width       *int   `json:"width"`
	Height      *int   `json:"height"`
}

type PhotoUploadResponse struct {
	Photo   PhotoResponse              `json:"photo"`
	Uploads PhotoUploadTargetsResponse `json:"uploads"`
}

type PhotoUploadTargetsResponse struct {
	Full    PhotoUploadTargetResponse `json:"full"`
	Display PhotoUploadTargetResponse `json:"display"`
}

type PhotoUploadTargetResponse struct {
	ObjectKey string    `json:"objectKey"`
	SignedURL string    `json:"signedUrl"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	PublicURL string    `json:"publicUrl"`
}

type InventoryResponse struct {
	Products []ProductResponse `json:"products"`
}

type ProductResponse struct {
	ID        string                   `json:"id"`
	BandID    string                   `json:"bandId"`
	Name      string                   `json:"name"`
	Category  inventorydomain.Category `json:"category"`
	Photo     PhotoResponse            `json:"photo"`
	Variants  []VariantResponse        `json:"variants"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

type VariantResponse struct {
	ID        string               `json:"id"`
	ProductID string               `json:"productId"`
	Size      inventorydomain.Size `json:"size"`
	Colour    string               `json:"colour"`
	Price     MoneyResponse        `json:"price"`
	Cost      MoneyResponse        `json:"cost"`
	Quantity  int                  `json:"quantity"`
	SoldOut   bool                 `json:"soldOut"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

type MoneyResponse struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type PhotoResponse struct {
	Full    PhotoVariantResponse `json:"full"`
	Display PhotoVariantResponse `json:"display"`
}

type PhotoVariantResponse struct {
	ObjectKey   string `json:"objectKey"`
	ContentType string `json:"contentType"`
	SizeBytes   int    `json:"sizeBytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PublicURL   string `json:"publicUrl"`
}

func NewHandler(inventoryRepository applicationinventory.Repository, photoStorage applicationinventory.PhotoStorage, logger *slog.Logger) Handler {
	return Handler{
		inventoryRepository: inventoryRepository,
		photoStorage:        photoStorage,
		logger:              logger,
		now:                 time.Now,
	}
}

func (handler Handler) CreatePhotoUpload(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	var body PhotoUploadRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	uploadRequest, err := applicationinventory.CreatePhotoUpload(request.Context(), handler.photoStorage, applicationinventory.CreatePhotoUploadInput{
		Account: accountContext,
		Full:    toPhotoUploadVariantInput(body.Full),
		Display: toPhotoUploadVariantInput(body.Display),
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory photo upload request failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toPhotoUploadResponse(handler.photoStorage, uploadRequest))
}

func (handler Handler) CreateProduct(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body CreateProductRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	variants, ok := toVariantInputs(response, body.Variants)
	if !ok {
		return
	}

	product, err := applicationinventory.CreateProduct(request.Context(), handler.inventoryRepository, handler.photoStorage, applicationinventory.CreateProductInput{
		Account:        accountContext,
		Name:           body.Name,
		Category:       body.Category,
		Photo:          toPhotoInput(body.Photo),
		Variants:       variants,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory create failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toProductResponse(handler.photoStorage, product))
}

func (handler Handler) CreateVariant(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body VariantRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	variant, ok := toVariantInput(response, body)
	if !ok {
		return
	}

	createdVariant, err := applicationinventory.CreateVariant(request.Context(), handler.inventoryRepository, applicationinventory.CreateVariantInput{
		Account:        accountContext,
		ProductID:      chi.URLParam(request, "productID"),
		Variant:        variant,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		CreatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory variant create failed", err)
		return
	}

	handler.writeJSON(response, http.StatusCreated, toVariantResponse(createdVariant))
}

func (handler Handler) ListInventory(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	products, err := applicationinventory.ListInventory(request.Context(), handler.inventoryRepository, applicationinventory.ListInventoryInput{
		Account: accountContext,
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory list failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, InventoryResponse{Products: toProductResponses(handler.photoStorage, products)})
}

func (handler Handler) UpdateProduct(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body ProductRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	product, err := applicationinventory.UpdateProduct(request.Context(), handler.inventoryRepository, handler.photoStorage, applicationinventory.UpdateProductInput{
		Account:        accountContext,
		ProductID:      chi.URLParam(request, "productID"),
		Name:           body.Name,
		Category:       body.Category,
		Photo:          toPhotoInput(body.Photo),
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		UpdatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory product update failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toProductResponse(handler.photoStorage, product))
}

func (handler Handler) UpdateVariant(response http.ResponseWriter, request *http.Request) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	var body VariantRequest
	if !handler.decodeJSON(response, request, &body) {
		return
	}

	variant, ok := toVariantInput(response, body)
	if !ok {
		return
	}

	updatedVariant, err := applicationinventory.UpdateVariant(request.Context(), handler.inventoryRepository, applicationinventory.UpdateVariantInput{
		Account:        accountContext,
		VariantID:      chi.URLParam(request, "variantID"),
		Variant:        variant,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		UpdatedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory variant update failed", err)
		return
	}

	handler.writeJSON(response, http.StatusOK, toVariantResponse(updatedVariant))
}

func (handler Handler) SoftDeleteProduct(response http.ResponseWriter, request *http.Request) {
	handler.softDeleteInventory(response, request, chi.URLParam(request, "productID"), applicationinventory.SoftDeleteProduct)
}

func (handler Handler) SoftDeleteVariant(response http.ResponseWriter, request *http.Request) {
	handler.softDeleteInventory(response, request, chi.URLParam(request, "variantID"), applicationinventory.SoftDeleteVariant)
}

func (handler Handler) softDeleteInventory(response http.ResponseWriter, request *http.Request, entityID string, deleteFunc func(context.Context, applicationinventory.Repository, applicationinventory.DeleteInventoryInput) error) {
	accountContext, ok := handler.accountContext(response, request)
	if !ok {
		return
	}

	idempotencyKey, requestID, ok := handler.mutationHeaders(response, request)
	if !ok {
		return
	}

	err := deleteFunc(request.Context(), handler.inventoryRepository, applicationinventory.DeleteInventoryInput{
		Account:        accountContext,
		EntityID:       entityID,
		IdempotencyKey: idempotencyKey,
		RequestID:      requestID,
		DeletedAt:      handler.now().UTC(),
	})
	if err != nil {
		handler.writeInventoryError(response, "inventory delete failed", err)
		return
	}

	response.WriteHeader(http.StatusNoContent)
}

func (handler Handler) accountContext(response http.ResponseWriter, request *http.Request) (applicationinventory.AccountContext, bool) {
	context, ok := authcontext.FromContext(request.Context())
	if !ok {
		handler.writeError(response, http.StatusInternalServerError, "missing_account_context", "Account context is missing")
		return applicationinventory.AccountContext{}, false
	}

	return applicationinventory.AccountContext{
		UserID: context.UserID,
		BandID: context.BandID,
		Role:   context.Role,
	}, true
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

func (handler Handler) writeInventoryError(response http.ResponseWriter, logMessage string, err error) {
	handler.logger.Warn(logMessage, "error", err)

	switch {
	case errors.Is(err, applicationinventory.ErrDuplicateProduct):
		handler.writeError(response, http.StatusConflict, "duplicate_product", err.Error())
	case errors.Is(err, applicationinventory.ErrDuplicateVariant):
		handler.writeError(response, http.StatusConflict, "duplicate_variant", err.Error())
	case errors.Is(err, applicationinventory.ErrLastInventoryVariant):
		handler.writeError(response, http.StatusConflict, "last_inventory_variant", err.Error())
	case errors.Is(err, applicationinventory.ErrInventoryNotFound):
		handler.writeError(response, http.StatusNotFound, "inventory_not_found", err.Error())
	case strings.Contains(err.Error(), "alpha write access requires owner role"):
		handler.writeError(response, http.StatusForbidden, "write_forbidden", err.Error())
	default:
		handler.writeError(response, http.StatusBadRequest, "invalid_inventory_request", err.Error())
	}
}

func (handler Handler) writeJSON(response http.ResponseWriter, statusCode int, body interface{}) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(statusCode)

	if err := json.NewEncoder(response).Encode(body); err != nil {
		handler.logger.Error("inventory response encoding failed", "error", err)
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
		handler.logger.Error("inventory error response encoding failed", "error", err, "code", code, "status_code", statusCode)
	}
}

func toVariantInputs(response http.ResponseWriter, requests []VariantRequest) ([]applicationinventory.VariantInput, bool) {
	variants := make([]applicationinventory.VariantInput, 0, len(requests))
	for _, request := range requests {
		variant, ok := toVariantInput(response, request)
		if !ok {
			return nil, false
		}
		variants = append(variants, variant)
	}

	return variants, true
}

func toVariantInput(response http.ResponseWriter, request VariantRequest) (applicationinventory.VariantInput, bool) {
	if request.Price.Amount == nil {
		writeRequestError(response, "missing_price_amount", "price.amount is required")
		return applicationinventory.VariantInput{}, false
	}

	if request.Cost.Amount == nil {
		writeRequestError(response, "missing_cost_amount", "cost.amount is required")
		return applicationinventory.VariantInput{}, false
	}

	if request.Quantity == nil {
		writeRequestError(response, "missing_quantity", "quantity is required")
		return applicationinventory.VariantInput{}, false
	}

	if request.Price.Currency != request.Cost.Currency {
		writeRequestError(response, "currency_mismatch", "price.currency and cost.currency must match")
		return applicationinventory.VariantInput{}, false
	}

	return applicationinventory.VariantInput{
		Size:        request.Size,
		Colour:      request.Colour,
		PriceAmount: *request.Price.Amount,
		CostAmount:  *request.Cost.Amount,
		Currency:    request.Price.Currency,
		Quantity:    *request.Quantity,
	}, true
}

func toPhotoInput(request PhotoRequest) applicationinventory.PhotoInput {
	return applicationinventory.PhotoInput{
		Full:    toPhotoVariantInput(request.Full),
		Display: toPhotoVariantInput(request.Display),
	}
}

func toPhotoVariantInput(request PhotoVariantRequest) applicationinventory.PhotoVariantInput {
	return applicationinventory.PhotoVariantInput{
		ObjectKey:   request.ObjectKey,
		ContentType: request.ContentType,
		SizeBytes:   intPointerValue(request.SizeBytes),
		Width:       intPointerValue(request.Width),
		Height:      intPointerValue(request.Height),
	}
}

func toPhotoUploadVariantInput(request PhotoUploadVariantRequest) applicationinventory.PhotoUploadVariantInput {
	return applicationinventory.PhotoUploadVariantInput{
		ContentType: request.ContentType,
		SizeBytes:   intPointerValue(request.SizeBytes),
		Width:       intPointerValue(request.Width),
		Height:      intPointerValue(request.Height),
	}
}

func intPointerValue(value *int) int {
	if value == nil {
		return 0
	}

	return *value
}

func toProductResponses(photoStorage applicationinventory.PhotoStorage, products []applicationinventory.Product) []ProductResponse {
	responses := make([]ProductResponse, 0, len(products))
	for _, product := range products {
		responses = append(responses, toProductResponse(photoStorage, product))
	}

	return responses
}

func toProductResponse(photoStorage applicationinventory.PhotoStorage, product applicationinventory.Product) ProductResponse {
	return ProductResponse{
		ID:        product.ID,
		BandID:    product.BandID,
		Name:      product.Name,
		Category:  product.Category,
		Photo:     toPhotoResponse(photoStorage, product.Photo),
		Variants:  toVariantResponses(product.Variants),
		CreatedAt: product.CreatedAt,
		UpdatedAt: product.UpdatedAt,
	}
}

func toPhotoUploadResponse(photoStorage applicationinventory.PhotoStorage, uploadRequest applicationinventory.PhotoUploadRequest) PhotoUploadResponse {
	return PhotoUploadResponse{
		Photo: toPhotoResponse(photoStorage, uploadRequest.Photo),
		Uploads: PhotoUploadTargetsResponse{
			Full:    toPhotoUploadTargetResponse(uploadRequest.Full),
			Display: toPhotoUploadTargetResponse(uploadRequest.Display),
		},
	}
}

func toPhotoUploadTargetResponse(upload applicationinventory.SignedPhotoUpload) PhotoUploadTargetResponse {
	return PhotoUploadTargetResponse{
		ObjectKey: upload.ObjectKey,
		SignedURL: upload.SignedURL,
		Token:     upload.Token,
		ExpiresAt: upload.ExpiresAt,
		PublicURL: upload.PublicURL,
	}
}

func toPhotoResponse(photoStorage applicationinventory.PhotoStorage, photo inventorydomain.PhotoMetadata) PhotoResponse {
	return PhotoResponse{
		Full:    toPhotoVariantResponse(photoStorage, photo.Full),
		Display: toPhotoVariantResponse(photoStorage, photo.Display),
	}
}

func toPhotoVariantResponse(photoStorage applicationinventory.PhotoStorage, photo inventorydomain.PhotoVariantMetadata) PhotoVariantResponse {
	return PhotoVariantResponse{
		ObjectKey:   photo.ObjectKey,
		ContentType: photo.ContentType,
		SizeBytes:   photo.SizeBytes,
		Width:       photo.Width,
		Height:      photo.Height,
		PublicURL:   photoStorage.PublicURL(photo.ObjectKey),
	}
}

func toVariantResponses(variants []applicationinventory.Variant) []VariantResponse {
	responses := make([]VariantResponse, 0, len(variants))
	for _, variant := range variants {
		responses = append(responses, toVariantResponse(variant))
	}

	return responses
}

func toVariantResponse(variant applicationinventory.Variant) VariantResponse {
	return VariantResponse{
		ID:        variant.ID,
		ProductID: variant.ProductID,
		Size:      variant.Size,
		Colour:    variant.Colour,
		Price: MoneyResponse{
			Amount:   variant.Price.Amount,
			Currency: variant.Price.Currency,
		},
		Cost: MoneyResponse{
			Amount:   variant.Cost.Amount,
			Currency: variant.Cost.Currency,
		},
		Quantity:  variant.Quantity,
		SoldOut:   variant.Quantity == 0,
		CreatedAt: variant.CreatedAt,
		UpdatedAt: variant.UpdatedAt,
	}
}

func writeRequestError(response http.ResponseWriter, code string, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(response).Encode(authhandler.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationcalendar "github.com/thalys/band-manager/apps/api/internal/application/calendar"
	applicationfinancialreports "github.com/thalys/band-manager/apps/api/internal/application/financialreports"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	applicationmerchbooth "github.com/thalys/band-manager/apps/api/internal/application/merchbooth"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/platform/config"
	"github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	calendarhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/calendar"
	financialreportshandler "github.com/thalys/band-manager/apps/api/internal/transport/http/financialreports"
	inventoryhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/inventory"
	merchboothhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/merchbooth"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Dependencies struct {
	Authenticator              session.Authenticator
	AccountRepository          accounts.BandAccountRepository
	InventoryRepository        applicationinventory.Repository
	MerchBoothRepository       applicationmerchbooth.Repository
	FinancialReportsRepository applicationfinancialreports.Repository
	CalendarRepository         applicationcalendar.Repository
	PaymentProvider            applicationmerchbooth.PaymentProvider
}

func NewRouter(appConfig config.Config, appLogger *slog.Logger, dependencies Dependencies) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.CORS(appConfig.AllowedOrigins))

	router.Get("/healthz", healthHandler(appLogger))
	authHandler := authhandler.NewHandler(dependencies.Authenticator, dependencies.AccountRepository, appLogger)
	router.Post("/auth/signup", authHandler.SignupOwner)
	router.Get("/me", authHandler.GetCurrentAccount)

	inventoryHandler := inventoryhandler.NewHandler(dependencies.Authenticator, dependencies.AccountRepository, dependencies.InventoryRepository, appLogger)
	router.Get("/inventory", inventoryHandler.ListInventory)
	router.Post("/inventory/products", inventoryHandler.CreateProduct)
	router.Put("/inventory/products/{productID}", inventoryHandler.UpdateProduct)
	router.Delete("/inventory/products/{productID}", inventoryHandler.SoftDeleteProduct)
	router.Put("/inventory/variants/{variantID}", inventoryHandler.UpdateVariant)
	router.Delete("/inventory/variants/{variantID}", inventoryHandler.SoftDeleteVariant)

	merchBoothHandler := merchboothhandler.NewHandler(dependencies.Authenticator, dependencies.AccountRepository, dependencies.MerchBoothRepository, dependencies.PaymentProvider, appConfig.MercadoPagoWebhookSecret, appConfig.MercadoPagoPointTerminalID, appLogger)
	router.Get("/merch-booth/items", merchBoothHandler.ListBoothItems)
	router.Post("/merch-booth/checkouts/cash", merchBoothHandler.CreateCashCheckout)
	router.Post("/merch-booth/checkouts/pix", merchBoothHandler.CreatePixCheckout)
	router.Post("/merch-booth/checkouts/card", merchBoothHandler.CreateCardCheckout)
	router.Post("/merch-booth/payments/{paymentID}/verify", merchBoothHandler.VerifyPixPayment)
	router.Post("/webhooks/mercadopago/orders", merchBoothHandler.HandleMercadoPagoOrderWebhook)

	financialReportsHandler := financialreportshandler.NewHandler(dependencies.Authenticator, dependencies.AccountRepository, dependencies.FinancialReportsRepository, appLogger)
	router.Get("/financial-reports", financialReportsHandler.GetFinancialReport)

	calendarHandler := calendarhandler.NewHandler(dependencies.Authenticator, dependencies.AccountRepository, dependencies.CalendarRepository, appLogger)
	router.Get("/calendar-events", calendarHandler.ListEvents)
	router.Post("/calendar-events", calendarHandler.CreateEvent)
	router.Get("/calendar-events/{eventID}", calendarHandler.GetEvent)
	router.Put("/calendar-events/{eventID}", calendarHandler.UpdateEvent)
	router.Delete("/calendar-events/{eventID}", calendarHandler.SoftDeleteEvent)

	return router
}

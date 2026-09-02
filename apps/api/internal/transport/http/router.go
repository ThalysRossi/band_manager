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
	accounthandler "github.com/thalys/band-manager/apps/api/internal/transport/http/account"
	"github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	calendarhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/calendar"
	financialreportshandler "github.com/thalys/band-manager/apps/api/internal/transport/http/financialreports"
	inventoryhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/inventory"
	merchboothhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/merchbooth"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Dependencies struct {
	Authenticator              session.Authenticator
	VerifiedUserInspector      session.VerifiedUserInspector
	AccountRepository          accounts.BandAccountRepository
	InventoryRepository        applicationinventory.Repository
	PhotoStorage               applicationinventory.PhotoStorage
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

	authHandler := authhandler.NewHandler(dependencies.AccountRepository, appLogger)
	accountHandler := accounthandler.NewHandler(dependencies.AccountRepository, appLogger)
	inventoryHandler := inventoryhandler.NewHandler(dependencies.InventoryRepository, dependencies.PhotoStorage, appLogger)
	merchBoothHandler := merchboothhandler.NewHandler(dependencies.MerchBoothRepository, dependencies.PaymentProvider, dependencies.PhotoStorage, appConfig.MercadoPagoWebhookSecret, appConfig.MercadoPagoPointTerminalID, appLogger)
	financialReportsHandler := financialreportshandler.NewHandler(dependencies.FinancialReportsRepository, appLogger)
	calendarHandler := calendarhandler.NewHandler(dependencies.CalendarRepository, appLogger)

	router.Post("/webhooks/mercadopago/orders", merchBoothHandler.HandleMercadoPagoOrderWebhook)

	router.Group(func(authenticated chi.Router) {
		authenticated.Use(middleware.Authenticate(dependencies.Authenticator, appLogger))

		authenticated.Group(func(verified chi.Router) {
			verified.Use(middleware.RequireVerifiedIdentity(dependencies.VerifiedUserInspector, appLogger))
			verified.Post("/account/onboarding", authHandler.OnboardOwner)
			verified.Post("/account/invites/accept", accountHandler.AcceptInvite)
		})

		authenticated.Group(func(protected chi.Router) {
			protected.Use(middleware.ResolveAccount(dependencies.AccountRepository, appLogger))
			protected.Get("/me", authHandler.GetCurrentAccount)
			protected.Get("/account/members", accountHandler.ListMembers)
			protected.Get("/account/invites", accountHandler.ListInvites)
			protected.Post("/account/invites", accountHandler.CreateInvite)
			protected.Post("/account/invites/{inviteID}/revoke", accountHandler.RevokeInvite)

			protected.Get("/inventory", inventoryHandler.ListInventory)
			protected.Post("/inventory/photos/upload-requests", inventoryHandler.CreatePhotoUpload)
			protected.Post("/inventory/products", inventoryHandler.CreateProduct)
			protected.Post("/inventory/products/{productID}/variants", inventoryHandler.CreateVariant)
			protected.Put("/inventory/products/{productID}", inventoryHandler.UpdateProduct)
			protected.Delete("/inventory/products/{productID}", inventoryHandler.SoftDeleteProduct)
			protected.Put("/inventory/variants/{variantID}", inventoryHandler.UpdateVariant)
			protected.Delete("/inventory/variants/{variantID}", inventoryHandler.SoftDeleteVariant)

			protected.Get("/merch-booth/items", merchBoothHandler.ListBoothItems)
			protected.Post("/merch-booth/checkouts/cash", merchBoothHandler.CreateCashCheckout)
			protected.Post("/merch-booth/checkouts/pix", merchBoothHandler.CreatePixCheckout)
			protected.Post("/merch-booth/checkouts/card", merchBoothHandler.CreateCardCheckout)
			protected.Post("/merch-booth/payments/{paymentID}/verify", merchBoothHandler.VerifyPixPayment)

			protected.Get("/financial-reports", financialReportsHandler.GetFinancialReport)
			protected.Get("/calendar-events", calendarHandler.ListEvents)
			protected.Post("/calendar-events", calendarHandler.CreateEvent)
			protected.Get("/calendar-events/{eventID}", calendarHandler.GetEvent)
			protected.Put("/calendar-events/{eventID}", calendarHandler.UpdateEvent)
			protected.Delete("/calendar-events/{eventID}", calendarHandler.SoftDeleteEvent)
		})
	})

	return router
}

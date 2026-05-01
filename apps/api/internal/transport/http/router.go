package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/thalys/band-manager/apps/api/internal/application/accounts"
	applicationinventory "github.com/thalys/band-manager/apps/api/internal/application/inventory"
	"github.com/thalys/band-manager/apps/api/internal/application/session"
	"github.com/thalys/band-manager/apps/api/internal/platform/config"
	"github.com/thalys/band-manager/apps/api/internal/transport/http/auth"
	inventoryhandler "github.com/thalys/band-manager/apps/api/internal/transport/http/inventory"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

type Dependencies struct {
	Authenticator       session.Authenticator
	AccountRepository   accounts.BandAccountRepository
	InventoryRepository applicationinventory.Repository
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

	return router
}

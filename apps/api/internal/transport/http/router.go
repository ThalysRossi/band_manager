package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/thalys/band-manager/apps/api/internal/platform/config"
	"github.com/thalys/band-manager/apps/api/internal/transport/middleware"
)

func NewRouter(appConfig config.Config, appLogger *slog.Logger) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.CORS(appConfig.AllowedOrigins))

	router.Get("/healthz", healthHandler(appLogger))

	return router
}

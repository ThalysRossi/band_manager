package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thalys/band-manager/apps/api/internal/infrastructure/mercadopago"
	postgresaccount "github.com/thalys/band-manager/apps/api/internal/infrastructure/postgres/account"
	postgresfinancialreports "github.com/thalys/band-manager/apps/api/internal/infrastructure/postgres/financialreports"
	postgresinventory "github.com/thalys/band-manager/apps/api/internal/infrastructure/postgres/inventory"
	postgresmerchbooth "github.com/thalys/band-manager/apps/api/internal/infrastructure/postgres/merchbooth"
	"github.com/thalys/band-manager/apps/api/internal/infrastructure/supabase"
	"github.com/thalys/band-manager/apps/api/internal/platform/config"
	"github.com/thalys/band-manager/apps/api/internal/platform/logger"
	httpapi "github.com/thalys/band-manager/apps/api/internal/transport/http"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appConfig, err := config.LoadFromEnvironment()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	appLogger := logger.New(appConfig.Environment)
	databasePool, err := pgxpool.New(ctx, appConfig.DatabaseURL)
	if err != nil {
		appLogger.Error("database pool creation failed", "error", err)
		os.Exit(1)
	}
	defer databasePool.Close()

	authenticator, err := supabase.NewAuthenticator(appConfig.SupabaseJWTSecret)
	if err != nil {
		appLogger.Error("supabase authenticator creation failed", "error", err)
		os.Exit(1)
	}

	accountRepository := postgresaccount.NewRepository(databasePool)
	inventoryRepository := postgresinventory.NewRepository(databasePool)
	merchBoothRepository := postgresmerchbooth.NewRepository(databasePool)
	financialReportsRepository := postgresfinancialreports.NewRepository(databasePool)
	paymentProvider, err := mercadopago.NewClient(appConfig.MercadoPagoAccessToken, "https://api.mercadopago.com", http.DefaultClient, appLogger)
	if err != nil {
		appLogger.Error("mercadopago client creation failed", "error", err)
		os.Exit(1)
	}
	server := &http.Server{
		Addr: appConfig.Address,
		Handler: httpapi.NewRouter(appConfig, appLogger, httpapi.Dependencies{
			Authenticator:              authenticator,
			AccountRepository:          accountRepository,
			InventoryRepository:        inventoryRepository,
			MerchBoothRepository:       merchBoothRepository,
			FinancialReportsRepository: financialReportsRepository,
			PaymentProvider:            paymentProvider,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		appLogger.Info("api server starting", "address", appConfig.Address, "environment", appConfig.Environment)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			appLogger.Error("api server failed", "error", serveErr)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}

	appLogger.Info("api server stopped")
}

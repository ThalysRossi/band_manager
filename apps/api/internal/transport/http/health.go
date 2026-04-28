package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

func healthHandler(appLogger *slog.Logger) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)

		body := healthResponse{Status: "ok"}
		if err := json.NewEncoder(response).Encode(body); err != nil {
			appLogger.Error("health response encoding failed", "error", err)
		}
	}
}

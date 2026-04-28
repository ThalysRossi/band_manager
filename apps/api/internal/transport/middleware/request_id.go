package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDContextKey struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		if requestID == "" {
			generatedRequestID, err := newRequestID()
			if err != nil {
				http.Error(response, "failed to generate request id", http.StatusInternalServerError)
				return
			}

			requestID = generatedRequestID
		}

		response.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(request.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok
}

func newRequestID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

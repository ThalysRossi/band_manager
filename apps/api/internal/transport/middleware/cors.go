package middleware

import "net/http"

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowedOriginSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowedOriginSet[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			origin := request.Header.Get("Origin")
			if _, ok := allowedOriginSet[origin]; ok {
				response.Header().Set("Access-Control-Allow-Origin", origin)
				response.Header().Set("Vary", "Origin")
				response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				response.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID")
				response.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			}

			if request.Method == http.MethodOptions {
				response.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(response, request)
		})
	}
}

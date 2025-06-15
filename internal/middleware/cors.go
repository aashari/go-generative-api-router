package middleware

import (
	"net/http"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// CORSMiddleware adds CORS headers to allow cross-origin requests
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set(utils.HeaderAccessControlAllowOrigin, utils.CORSAllowOriginAll)
		w.Header().Set(utils.HeaderAccessControlAllowMethods, utils.CORSAllowMethodsAll)
		w.Header().Set(utils.HeaderAccessControlAllowHeaders, utils.CORSAllowHeadersStd)

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Process the request
		next.ServeHTTP(w, r)
	})
}
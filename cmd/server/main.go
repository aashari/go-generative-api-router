package main

import (
	"net/http"
	"os"
	"time"

	"github.com/aashari/go-generative-api-router/internal/app"
	"github.com/aashari/go-generative-api-router/internal/logger"
)

// @title           Generative API Router
// @version         1.0
// @description     A router for generative AI models with OpenAI-compatible API.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://github.com/aashari/go-generative-api-router
// @contact.email  support@yourcompany.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      genapi.aduh.xyz
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the API key value.

// CORSMiddleware adds CORS headers to allow cross-origin requests
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Process the request
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize structured logging
	if err := logger.InitFromEnv(); err != nil {
		// Can't use logger here as it failed to initialize
		// Fall back to basic output and exit
		_, _ = os.Stderr.WriteString("FATAL: Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Create and initialize the application
	app, err := app.NewApp()
	if err != nil {
		logger.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Get router with all routes configured
	handler := app.SetupRoutes()

	// Apply CORS middleware
	corsHandler := CORSMiddleware(handler)

	logger.Info("Server starting", "address", "0.0.0.0:8082")
	logger.Info("Swagger documentation available", "url", "https://genapi.aduh.xyz/swagger/index.html")

	srv := &http.Server{
		Addr:         "0.0.0.0:8082",
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	err = srv.ListenAndServe()
	if err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

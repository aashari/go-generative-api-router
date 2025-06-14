package main

import (
	"context"
	"net/http"
	"os"

	"github.com/aashari/go-generative-api-router/internal/app"
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
)

// version is set at build time via ldflags
var version = "unknown"

// @title           Generative API Router
// @version         1.0
// @description     A router for generative AI models with OpenAI-compatible API.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://github.com/aashari/go-generative-api-router
// @contact.email  support@yourcompany.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      genapi.example.com
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
	// Set VERSION environment variable from build-time version if not already set
	if os.Getenv("VERSION") == "" {
		os.Setenv("VERSION", version)
	}

	// Load environment variables from .env file
	err := config.LoadEnvFile()
	if err != nil {
		logger.Warn(context.Background(), "No .env file found, using environment variables")
	}

	// Initialize logger
	logger.InitFromEnv()

	// Create a new application instance
	appInstance, err := app.NewApp()
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize application", err)
		os.Exit(1)
	}

	// Setup router
	r := appInstance.SetupRoutes()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	logger.Info(context.Background(), "Starting server", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error(context.Background(), "Failed to start server", err)
		os.Exit(1)
	}
}

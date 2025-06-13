package main

import (
	"net/http"
	"os"
	"time"

	"github.com/aashari/go-generative-api-router/internal/app"
	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/utils"
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

	// Load environment variables from .env file (similar to Node.js dotenv)
	if err := config.LoadEnvFromMultiplePaths(); err != nil {
		// This is not fatal - the app can run with system environment variables
		_, _ = os.Stderr.WriteString("WARNING: Could not load .env file: " + err.Error() + "\n")
	}

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

	// Configure server address and timeouts from environment variables
	serverAddr := utils.GetEnvString("SERVER_ADDR", "0.0.0.0:8082")
	if port := os.Getenv("PORT"); port != "" {
		serverAddr = "0.0.0.0:" + port
	}

	// Configure timeouts with generous defaults for AI workloads
	// Server timeouts should be longer than client timeout to prevent premature connection closure
	// Increased timeouts to prevent 120-second client timeouts
	readTimeout := utils.GetEnvDuration("READ_TIMEOUT", 1500*time.Second)   // 25 minutes default
	writeTimeout := utils.GetEnvDuration("WRITE_TIMEOUT", 1500*time.Second) // 25 minutes default
	idleTimeout := utils.GetEnvDuration("IDLE_TIMEOUT", 1800*time.Second)   // 30 minutes default

	logger.Info("Server starting",
		"address", serverAddr,
		"version", version,
		"read_timeout", readTimeout,
		"write_timeout", writeTimeout,
		"idle_timeout", idleTimeout,
	)
	logger.Info("Swagger documentation available", "url", "https://genapi.example.com/swagger/index.html")

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      corsHandler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	err = srv.ListenAndServe()
	if err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

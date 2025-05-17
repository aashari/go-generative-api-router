package main

import (
	"log"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/app"
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
	// Create and initialize the application
	app, err := app.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Get router with all routes configured
	handler := app.SetupRoutes()

	// Apply CORS middleware
	corsHandler := CORSMiddleware(handler)

	log.Println("Server starting on 0.0.0.0:8082")
	log.Println("Swagger documentation available at https://genapi.aduh.xyz/swagger/index.html")
	err = http.ListenAndServe("0.0.0.0:8082", corsHandler)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

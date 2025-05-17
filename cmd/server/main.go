package main

import (
	"log"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/app"
)

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

	mux := http.NewServeMux()

	// Register handlers from the app
	mux.HandleFunc("/health", app.HealthHandler)
	mux.HandleFunc("/chat/completions", app.ChatCompletionsHandler)
	mux.HandleFunc("/models", app.ModelsHandler)

	// Apply CORS middleware
	corsHandler := CORSMiddleware(mux)

	log.Println("Server starting on :8082")
	err = http.ListenAndServe(":8082", corsHandler)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

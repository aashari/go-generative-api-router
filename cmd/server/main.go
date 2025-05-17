package main

import (
	"log"
	"net/http"

	"github.com/aashari/go-generative-api-router/internal/app"
)

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

	log.Println("Server starting on :8082")
	err = http.ListenAndServe(":8082", mux)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

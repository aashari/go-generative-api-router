package main

import (
    "log"
    "math/rand"
    "net/http"
    "time"
    "github.com/aashari/generative-api-router/internal/config"
    "github.com/aashari/generative-api-router/internal/proxy"
)

func main() {
    // In Go 1.22, rand.Seed is deprecated
    // rand package now automatically seeds itself using a secure random number
    // For backwards compatibility, we'll set a random source explicitly
    rand.New(rand.NewSource(time.Now().UnixNano()))

    // Load credentials
    creds, err := config.LoadCredentials("credentials.json")
    if err != nil {
        log.Fatalf("Failed to load credentials: %v", err)
    }

    if len(creds) == 0 {
        log.Fatalf("No credentials found in configuration file")
    }
    
    // Load vendor-model pairs
    models, err := config.LoadVendorModels("models.json")
    if err != nil {
        log.Fatalf("Failed to load vendor-model pairs: %v", err)
    }
    
    if len(models) == 0 {
        log.Fatalf("No vendor-model pairs found in models.json")
    }
    
    log.Printf("Loaded %d credentials and %d vendor-model pairs", len(creds), len(models))

    mux := http.NewServeMux()
    mux.HandleFunc("/chat/completions", func(w http.ResponseWriter, r *http.Request) {
        proxy.ProxyRequest(w, r, creds, models)
    })

    log.Println("Server starting on :8082")
    err = http.ListenAndServe(":8082", mux)
    if err != nil {
        log.Fatalf("Server failed: %v", err)
    }
} 
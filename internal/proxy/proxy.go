package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"

	"github.com/aashari/generative-api-router/internal/config"
)

var baseURLs = map[string]string{
    "openai": "https://api.openai.com/v1",
    "gemini": "https://generativelanguage.googleapis.com/v1beta/openai",
}

func ProxyRequest(w http.ResponseWriter, r *http.Request, creds []config.Credential, models []config.VendorModel) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Randomly select a credential (which determines the vendor)
    if len(creds) == 0 {
        http.Error(w, "No credentials available", http.StatusInternalServerError)
        return
    }
    selectedCred := creds[rand.Intn(len(creds))]
    vendor := selectedCred.Platform
    
    log.Printf("Randomly selected credential for vendor: %s", vendor)

    // Filter models for the selected vendor
    var vendorModels []config.VendorModel
    for _, m := range models {
        if m.Vendor == vendor {
            vendorModels = append(vendorModels, m)
        }
    }
    
    if len(vendorModels) == 0 {
        http.Error(w, fmt.Sprintf("No models available for vendor: %s", vendor), http.StatusInternalServerError)
        return
    }

    // Randomly select a model for the vendor
    selectedModel := vendorModels[rand.Intn(len(vendorModels))]
    model := selectedModel.Model
    
    log.Printf("Randomly selected model: %s for vendor: %s", model, vendor)

    baseURL, ok := baseURLs[vendor]
    if !ok {
        http.Error(w, fmt.Sprintf("Unknown vendor: %s", vendor), http.StatusInternalServerError)
        return
    }

    // All vendors use the same OpenAI-compatible endpoint
    fullURL := baseURL + "/chat/completions"

    // Read the request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body: "+err.Error(), http.StatusBadRequest)
        return
    }
    r.Body.Close()

    // Parse the request to validate and modify it
    var requestData map[string]interface{}
    if err := json.Unmarshal(body, &requestData); err != nil {
        http.Error(w, "Invalid request format: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Validate messages exist
    if _, ok := requestData["messages"]; !ok {
        http.Error(w, "Missing 'messages' field in request", http.StatusBadRequest)
        return
    }

    // Replace the model with our selected one
    requestData["model"] = model

    // Re-encode the modified request
    modifiedBody, err := json.Marshal(requestData)
    if err != nil {
        http.Error(w, "Failed to encode modified request: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Create the proxied request
    req, err := http.NewRequest(r.Method, fullURL, bytes.NewReader(modifiedBody))
    if err != nil {
        http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Copy request headers
    for k, vs := range r.Header {
        for _, v := range vs {
            req.Header.Add(k, v)
        }
    }

    // Set authorization header using Bearer token for all vendors
    req.Header.Set("Authorization", "Bearer "+selectedCred.Value)

    // Send the request to the vendor
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to send request to vendor: %v", err), http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    // Copy response status code and headers
    w.WriteHeader(resp.StatusCode)
    for k, vs := range resp.Header {
        for _, v := range vs {
            w.Header().Add(k, v)
        }
    }

    // Stream the response directly to the client
    _, err = io.Copy(w, resp.Body)
    if err != nil {
        log.Printf("Error copying response: %v", err)
    }
} 
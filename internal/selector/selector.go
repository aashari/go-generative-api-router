package selector

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/aashari/go-generative-api-router/internal/config"
)

// VendorSelection stores the selected vendor, model and credential
type VendorSelection struct {
	Vendor     string
	Model      string
	Credential config.Credential
}

// RandomSelector is a selector that randomly chooses vendors and models
type RandomSelector struct {
	rng *rand.Rand
}

// NewRandomSelector creates a new random selector
func NewRandomSelector() *RandomSelector {
	// In Go 1.20+, math/rand is automatically seeded
	// For a shared selector, we could pass a custom rng for more control and testing
	return &RandomSelector{
		rng: rand.New(rand.NewSource(rand.Int63())), // This ensures each selector has its own randomness
	}
}

// Select randomly selects a vendor, model and its credential
func (s *RandomSelector) Select(creds []config.Credential, models []config.VendorModel) (*VendorSelection, error) {
	if len(creds) == 0 {
		return nil, fmt.Errorf("no credentials available")
	}

	selectedCred := creds[s.rng.Intn(len(creds))]
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
		return nil, fmt.Errorf("no models available for vendor: %s", vendor)
	}

	// Randomly select a model for the vendor
	selectedModel := vendorModels[s.rng.Intn(len(vendorModels))]
	model := selectedModel.Model

	log.Printf("Randomly selected model: %s for vendor: %s", model, vendor)

	return &VendorSelection{
		Vendor:     vendor,
		Model:      model,
		Credential: selectedCred,
	}, nil
}

// Selector interface for different selection strategies
type Selector interface {
	Select(creds []config.Credential, models []config.VendorModel) (*VendorSelection, error)
}

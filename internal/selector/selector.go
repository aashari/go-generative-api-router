package selector

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/aashari/generative-api-router/internal/config"
)

// VendorSelection stores the selected vendor, model and credential
type VendorSelection struct {
	Vendor     string
	Model      string
	Credential config.Credential
}

// RandomSelector is a selector that randomly chooses vendors and models
type RandomSelector struct {}

// NewRandomSelector creates a new random selector
func NewRandomSelector() *RandomSelector {
	return &RandomSelector{}
}

// Select randomly selects a vendor, model and its credential
func (s *RandomSelector) Select(creds []config.Credential, models []config.VendorModel) (*VendorSelection, error) {
	if len(creds) == 0 {
		return nil, fmt.Errorf("no credentials available")
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
		return nil, fmt.Errorf("no models available for vendor: %s", vendor)
	}

	// Randomly select a model for the vendor
	selectedModel := vendorModels[rand.Intn(len(vendorModels))]
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
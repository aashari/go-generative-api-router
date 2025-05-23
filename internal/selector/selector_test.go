package selector

import (
	"math/rand"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRandomSelector(t *testing.T) {
	selector := NewRandomSelector()
	require.NotNil(t, selector)
	require.NotNil(t, selector.rng)
}

func TestRandomSelector_Select(t *testing.T) {
	tests := []struct {
		name    string
		creds   []config.Credential
		models  []config.VendorModel
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful selection",
			creds: []config.Credential{
				{Platform: "openai", Type: "api-key", Value: "sk-test"},
				{Platform: "gemini", Type: "api-key", Value: "gemini-test"},
			},
			models: []config.VendorModel{
				{Vendor: "openai", Model: "gpt-4"},
				{Vendor: "openai", Model: "gpt-3.5-turbo"},
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			wantErr: false,
		},
		{
			name:    "no credentials",
			creds:   []config.Credential{},
			models:  []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}},
			wantErr: true,
			errMsg:  "no credentials available",
		},
		{
			name:  "no models for selected vendor",
			creds: []config.Credential{{Platform: "openai", Type: "api-key", Value: "sk-test"}},
			models: []config.VendorModel{
				{Vendor: "gemini", Model: "gemini-pro"},
			},
			wantErr: true,
			errMsg:  "no models available for vendor: openai",
		},
		{
			name:    "no models at all",
			creds:   []config.Credential{{Platform: "openai", Type: "api-key", Value: "sk-test"}},
			models:  []config.VendorModel{},
			wantErr: true,
			errMsg:  "no models available for vendor: openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewRandomSelector()
			selection, err := selector.Select(tt.creds, tt.models)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, selection)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, selection)

			// Verify selection is valid
			assert.NotEmpty(t, selection.Vendor)
			assert.NotEmpty(t, selection.Model)
			assert.NotEmpty(t, selection.Credential.Platform)
			assert.NotEmpty(t, selection.Credential.Value)

			// Verify vendor consistency
			assert.Equal(t, selection.Vendor, selection.Credential.Platform)

			// Verify model exists for the vendor
			found := false
			for _, model := range tt.models {
				if model.Vendor == selection.Vendor && model.Model == selection.Model {
					found = true
					break
				}
			}
			assert.True(t, found, "Selected model should exist for the vendor")

			// Verify credential exists
			credFound := false
			for _, cred := range tt.creds {
				if cred.Platform == selection.Credential.Platform {
					credFound = true
					break
				}
			}
			assert.True(t, credFound, "Selected credential should exist")
		})
	}
}

func TestRandomSelector_Randomness(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "sk-test1"},
		{Platform: "gemini", Type: "api-key", Value: "gemini-test1"},
		{Platform: "anthropic", Type: "api-key", Value: "anthropic-test1"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "openai", Model: "gpt-3.5-turbo"},
		{Vendor: "gemini", Model: "gemini-pro"},
		{Vendor: "anthropic", Model: "claude-3"},
	}

	selector := NewRandomSelector()
	
	// Run multiple selections to verify randomness
	selections := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		selection, err := selector.Select(creds, models)
		require.NoError(t, err)
		
		key := selection.Vendor + ":" + selection.Model
		selections[key]++
	}

	// With 3 vendors and 4 models, we should see some variety
	// This is a probabilistic test, so we're lenient
	assert.Greater(t, len(selections), 1, "Should select different vendor/model combinations")
}

func TestRandomSelector_SingleOption(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "sk-test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
	}

	selector := NewRandomSelector()
	selection, err := selector.Select(creds, models)

	require.NoError(t, err)
	assert.Equal(t, "openai", selection.Vendor)
	assert.Equal(t, "gpt-4", selection.Model)
	assert.Equal(t, "openai", selection.Credential.Platform)
	assert.Equal(t, "sk-test", selection.Credential.Value)
}

func TestRandomSelector_MultipleModelsForVendor(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "sk-test"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "openai", Model: "gpt-3.5-turbo"},
		{Vendor: "openai", Model: "gpt-4-turbo"},
	}

	selector := NewRandomSelector()
	
	// Run multiple selections to verify model randomness within vendor
	modelSelections := make(map[string]int)
	iterations := 50

	for i := 0; i < iterations; i++ {
		selection, err := selector.Select(creds, models)
		require.NoError(t, err)
		assert.Equal(t, "openai", selection.Vendor)
		modelSelections[selection.Model]++
	}

	// Should see some variety in model selection
	assert.Greater(t, len(modelSelections), 1, "Should select different models for the same vendor")
}

func TestRandomSelector_InterfaceCompliance(t *testing.T) {
	var _ Selector = &RandomSelector{}
}

func TestRandomSelector_DeterministicWithSeed(t *testing.T) {
	creds := []config.Credential{
		{Platform: "openai", Type: "api-key", Value: "sk-test1"},
		{Platform: "gemini", Type: "api-key", Value: "gemini-test1"},
	}
	models := []config.VendorModel{
		{Vendor: "openai", Model: "gpt-4"},
		{Vendor: "gemini", Model: "gemini-pro"},
	}

	// Create two selectors with same seed
	selector1 := &RandomSelector{rng: rand.New(rand.NewSource(42))}
	selector2 := &RandomSelector{rng: rand.New(rand.NewSource(42))}

	selection1, err1 := selector1.Select(creds, models)
	selection2, err2 := selector2.Select(creds, models)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// With same seed, selections should be identical
	assert.Equal(t, selection1.Vendor, selection2.Vendor)
	assert.Equal(t, selection1.Model, selection2.Model)
	assert.Equal(t, selection1.Credential.Platform, selection2.Credential.Platform)
}

func TestVendorSelection_Complete(t *testing.T) {
	selection := &VendorSelection{
		Vendor: "openai",
		Model:  "gpt-4",
		Credential: config.Credential{
			Platform: "openai",
			Type:     "api-key",
			Value:    "sk-test",
		},
	}

	assert.Equal(t, "openai", selection.Vendor)
	assert.Equal(t, "gpt-4", selection.Model)
	assert.Equal(t, "openai", selection.Credential.Platform)
	assert.Equal(t, "api-key", selection.Credential.Type)
	assert.Equal(t, "sk-test", selection.Credential.Value)
} 
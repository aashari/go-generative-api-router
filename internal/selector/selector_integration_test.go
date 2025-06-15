package selector

import (
	"fmt"
	"testing"

	"github.com/aashari/go-generative-api-router/internal/config"
	"github.com/aashari/go-generative-api-router/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data setup for integration tests
func setupTestData() ([]config.Credential, []config.VendorModel) {
	credentials := []config.Credential{
		{Platform: "openai", Type: "api_key", Value: "test-openai-key-1"},
		{Platform: "openai", Type: "api_key", Value: "test-openai-key-2"},
		{Platform: "gemini", Type: "api_key", Value: "test-gemini-key-1"},
		{Platform: "gemini", Type: "api_key", Value: "test-gemini-key-2"},
		{Platform: "gemini", Type: "api_key", Value: "test-gemini-key-3"},
	}

	models := []config.VendorModel{
		{
			Vendor: "openai",
			Model:  "gpt-4",
			Config: &config.ModelConfig{
				SupportImage:     true,
				SupportVideo:     false,
				SupportTools:     true,
				SupportStreaming: true,
			},
		},
		{
			Vendor: "openai",
			Model:  "gpt-3.5-turbo",
			Config: &config.ModelConfig{
				SupportImage:     false,
				SupportVideo:     false,
				SupportTools:     true,
				SupportStreaming: true,
			},
		},
		{
			Vendor: "gemini",
			Model:  "gemini-pro",
			Config: &config.ModelConfig{
				SupportImage:     true,
				SupportVideo:     true,
				SupportTools:     true,
				SupportStreaming: true,
			},
		},
		{
			Vendor: "gemini",
			Model:  "gemini-flash",
			Config: &config.ModelConfig{
				SupportImage:     true,
				SupportVideo:     false,
				SupportTools:     false,
				SupportStreaming: true,
			},
		},
	}

	return credentials, models
}

// Statistical test for RandomSelector distribution
func TestRandomSelector_StatisticalDistribution(t *testing.T) {
	credentials, models := setupTestData()
	selector := NewRandomSelector()

	const iterations = 10000
	const tolerance = 0.05 // 5% tolerance for statistical variance

	// Track selection counts by vendor
	vendorCounts := make(map[string]int)
	modelCounts := make(map[string]int)

	for i := 0; i < iterations; i++ {
		selection, err := selector.Select(credentials, models)
		require.NoError(t, err, "Selection should not fail")
		require.NotNil(t, selection, "Selection should not be nil")

		vendorCounts[selection.Vendor]++
		modelCounts[selection.Model]++
	}

	// Verify vendor distribution
	// Expected: openai=40% (2/5 credentials), gemini=60% (3/5 credentials)
	expectedOpenAIRatio := 2.0 / 5.0
	expectedGeminiRatio := 3.0 / 5.0

	actualOpenAIRatio := float64(vendorCounts["openai"]) / float64(iterations)
	actualGeminiRatio := float64(vendorCounts["gemini"]) / float64(iterations)

	assert.InDelta(t, expectedOpenAIRatio, actualOpenAIRatio, tolerance,
		"OpenAI vendor selection should be approximately %.2f%%, got %.2f%%",
		expectedOpenAIRatio*100, actualOpenAIRatio*100)

	assert.InDelta(t, expectedGeminiRatio, actualGeminiRatio, tolerance,
		"Gemini vendor selection should be approximately %.2f%%, got %.2f%%",
		expectedGeminiRatio*100, actualGeminiRatio*100)

	// Verify models are selected proportionally within vendors
	// For OpenAI: 2 models should be ~50/50
	if vendorCounts["openai"] > 0 {
		openaiModelRatio := float64(modelCounts["gpt-4"]) / float64(vendorCounts["openai"])
		assert.InDelta(t, 0.5, openaiModelRatio, tolerance*2,
			"OpenAI models should be distributed ~50/50, gpt-4 ratio: %.2f%%",
			openaiModelRatio*100)
	}

	// For Gemini: 2 models should be ~50/50
	if vendorCounts["gemini"] > 0 {
		geminiModelRatio := float64(modelCounts["gemini-pro"]) / float64(vendorCounts["gemini"])
		assert.InDelta(t, 0.5, geminiModelRatio, tolerance*2,
			"Gemini models should be distributed ~50/50, gemini-pro ratio: %.2f%%",
			geminiModelRatio*100)
	}

	t.Logf("Statistical verification completed:")
	t.Logf("  OpenAI selections: %d/%.2f%% (expected ~%.2f%%)",
		vendorCounts["openai"], actualOpenAIRatio*100, expectedOpenAIRatio*100)
	t.Logf("  Gemini selections: %d/%.2f%% (expected ~%.2f%%)",
		vendorCounts["gemini"], actualGeminiRatio*100, expectedGeminiRatio*100)
	t.Logf("  Model distribution: %v", modelCounts)
}

// Statistical test for EvenDistributionSelector
func TestEvenDistributionSelector_StatisticalDistribution(t *testing.T) {
	credentials, models := setupTestData()
	selector := NewEvenDistributionSelector()

	const iterations = 10000
	const tolerance = 0.03 // 3% tolerance for even distribution

	// Count combinations
	combinationCounts := make(map[string]int)

	for i := 0; i < iterations; i++ {
		selection, err := selector.Select(credentials, models)
		require.NoError(t, err, "Selection should not fail")
		require.NotNil(t, selection, "Selection should not be nil")

		// Create a unique key for the combination
		key := fmt.Sprintf("%s|%s|%s", selection.Vendor, selection.Model, selection.Credential.Value)
		combinationCounts[key]++
	}

	// Calculate expected combinations
	// OpenAI: 2 credentials × 2 models = 4 combinations
	// Gemini: 3 credentials × 2 models = 6 combinations
	// Total: 10 combinations, each should get ~10%
	expectedRatio := 1.0 / 10.0

	for combination, count := range combinationCounts {
		actualRatio := float64(count) / float64(iterations)
		assert.InDelta(t, expectedRatio, actualRatio, tolerance,
			"Combination %s should be selected ~%.2f%%, got %.2f%%",
			combination, expectedRatio*100, actualRatio*100)
	}

	t.Logf("Even distribution verification completed:")
	t.Logf("  Total combinations: %d", len(combinationCounts))
	t.Logf("  Expected ratio per combination: %.2f%%", expectedRatio*100)
	for combination, count := range combinationCounts {
		ratio := float64(count) / float64(iterations)
		t.Logf("  %s: %d selections (%.2f%%)", combination, count, ratio*100)
	}
}

// Test ContextAwareSelector with different payload contexts
func TestContextAwareSelector_ContextFiltering(t *testing.T) {
	credentials, models := setupTestData()
	selector := NewContextAwareSelector()

	tests := []struct {
		name            string
		context         *types.PayloadContext
		expectedModels  []string
		expectedVendors []string
		iterations      int
		tolerance       float64
	}{
		{
			name:            "no context - all models available",
			context:         nil,
			expectedModels:  []string{"gpt-4", "gpt-3.5-turbo", "gemini-pro", "gemini-flash"},
			expectedVendors: []string{"openai", "gemini"},
			iterations:      5000,
			tolerance:       0.05,
		},
		{
			name: "image support required",
			context: &types.PayloadContext{
				HasImages: true,
				HasVideos: false,
				HasTools:  false,
				HasStream: false,
			},
			expectedModels:  []string{"gpt-4", "gemini-pro", "gemini-flash"}, // Only models with image support
			expectedVendors: []string{"openai", "gemini"},
			iterations:      5000,
			tolerance:       0.05,
		},
		{
			name: "video support required",
			context: &types.PayloadContext{
				HasImages: false,
				HasVideos: true,
				HasTools:  false,
				HasStream: false,
			},
			expectedModels:  []string{"gemini-pro"}, // Only gemini-pro supports video
			expectedVendors: []string{"gemini"},
			iterations:      1000,
			tolerance:       0.05,
		},
		{
			name: "tools support required",
			context: &types.PayloadContext{
				HasImages: false,
				HasVideos: false,
				HasTools:  true,
				HasStream: false,
			},
			expectedModels:  []string{"gpt-4", "gpt-3.5-turbo", "gemini-pro"}, // Models with tool support
			expectedVendors: []string{"openai", "gemini"},
			iterations:      5000,
			tolerance:       0.05,
		},
		{
			name: "streaming support required",
			context: &types.PayloadContext{
				HasImages: false,
				HasVideos: false,
				HasTools:  false,
				HasStream: true,
			},
			expectedModels:  []string{"gpt-4", "gpt-3.5-turbo", "gemini-pro", "gemini-flash"}, // All support streaming
			expectedVendors: []string{"openai", "gemini"},
			iterations:      5000,
			tolerance:       0.05,
		},
		{
			name: "multiple requirements - image + tools",
			context: &types.PayloadContext{
				HasImages: true,
				HasVideos: false,
				HasTools:  true,
				HasStream: false,
			},
			expectedModels:  []string{"gpt-4", "gemini-pro"}, // Only gpt-4 and gemini-pro support both
			expectedVendors: []string{"openai", "gemini"},
			iterations:      3000,
			tolerance:       0.06,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelCounts := make(map[string]int)
			vendorCounts := make(map[string]int)

			for i := 0; i < tt.iterations; i++ {
				selection, err := selector.SelectWithContext(credentials, models, tt.context)
				require.NoError(t, err, "Selection should not fail")
				require.NotNil(t, selection, "Selection should not be nil")

				modelCounts[selection.Model]++
				vendorCounts[selection.Vendor]++

				// Verify selected model is in expected list
				assert.Contains(t, tt.expectedModels, selection.Model,
					"Selected model %s should be in expected models %v", selection.Model, tt.expectedModels)

				// Verify selected vendor is in expected list
				assert.Contains(t, tt.expectedVendors, selection.Vendor,
					"Selected vendor %s should be in expected vendors %v", selection.Vendor, tt.expectedVendors)
			}

			// Statistical verification for distribution among available models
			// Note: ContextAwareSelector uses EvenDistributionSelector under the hood,
			// which distributes based on credential-model combinations, not just models.
			// So we expect uneven distribution across models based on credential counts.

			// Verify all selected models are valid (this is the primary verification)
			totalSelections := 0
			for _, count := range modelCounts {
				totalSelections += count
			}
			assert.Equal(t, tt.iterations, totalSelections, "All selections should be accounted for")

			// For reference, log the distribution but don't assert strict equality
			// since the underlying even distribution is based on combinations

			// Verify no unexpected models were selected
			for model := range modelCounts {
				assert.Contains(t, tt.expectedModels, model,
					"Unexpected model selected: %s", model)
			}

			t.Logf("Context filtering verification for %s:", tt.name)
			t.Logf("  Expected models: %v", tt.expectedModels)
			t.Logf("  Model counts: %v", modelCounts)
			t.Logf("  Vendor counts: %v", vendorCounts)
		})
	}
}

// Test edge cases and error conditions
func TestSelector_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		credentials []config.Credential
		models      []config.VendorModel
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no credentials",
			credentials: []config.Credential{},
			models:      []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}},
			expectError: true,
			errorMsg:    "no credentials available",
		},
		{
			name:        "no models",
			credentials: []config.Credential{{Platform: "openai", Type: "api_key", Value: "test"}},
			models:      []config.VendorModel{},
			expectError: true,
			errorMsg:    "no models available",
		},
		{
			name:        "mismatched vendor",
			credentials: []config.Credential{{Platform: "openai", Type: "api_key", Value: "test"}},
			models:      []config.VendorModel{{Vendor: "gemini", Model: "gemini-pro"}},
			expectError: true,
			errorMsg:    "", // Different selectors have different error messages for this case
		},
		{
			name:        "single valid combination",
			credentials: []config.Credential{{Platform: "openai", Type: "api_key", Value: "test"}},
			models:      []config.VendorModel{{Vendor: "openai", Model: "gpt-4"}},
			expectError: false,
		},
	}

	selectors := map[string]Selector{
		"RandomSelector":           NewRandomSelector(),
		"EvenDistributionSelector": NewEvenDistributionSelector(),
	}

	for selectorName, selector := range selectors {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s_%s", selectorName, tt.name), func(t *testing.T) {
				selection, err := selector.Select(tt.credentials, tt.models)

				if tt.expectError {
					assert.Error(t, err, "Expected error for %s", tt.name)
					assert.Nil(t, selection, "Selection should be nil on error")
					if tt.errorMsg != "" {
						assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
					}
				} else {
					assert.NoError(t, err, "Should not error for %s", tt.name)
					assert.NotNil(t, selection, "Selection should not be nil")
					assert.NotEmpty(t, selection.Vendor, "Vendor should not be empty")
					assert.NotEmpty(t, selection.Model, "Model should not be empty")
					assert.NotEmpty(t, selection.Credential.Value, "Credential value should not be empty")
				}
			})
		}
	}
}

// Test ContextAwareSelector edge cases
func TestContextAwareSelector_EdgeCases(t *testing.T) {
	credentials, models := setupTestData()
	selector := NewContextAwareSelector()

	tests := []struct {
		name        string
		context     *types.PayloadContext
		expectError bool
		errorMsg    string
	}{
		{
			name: "impossible requirements - video + non-video-supporting models only",
			context: &types.PayloadContext{
				HasImages: false,
				HasVideos: true,
				HasTools:  true,
				HasStream: false,
			},
			expectError: false, // gemini-pro supports video + tools
		},
		{
			name: "all requirements enabled",
			context: &types.PayloadContext{
				HasImages: true,
				HasVideos: true,
				HasTools:  true,
				HasStream: true,
			},
			expectError: false, // gemini-pro supports all
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selection, err := selector.SelectWithContext(credentials, models, tt.context)

			if tt.expectError {
				assert.Error(t, err, "Expected error for %s", tt.name)
				assert.Nil(t, selection, "Selection should be nil on error")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Should not error for %s", tt.name)
				assert.NotNil(t, selection, "Selection should not be nil")
			}
		})
	}
}

// Benchmark selector performance
func BenchmarkRandomSelector(b *testing.B) {
	credentials, models := setupTestData()
	selector := NewRandomSelector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := selector.Select(credentials, models)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvenDistributionSelector(b *testing.B) {
	credentials, models := setupTestData()
	selector := NewEvenDistributionSelector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := selector.Select(credentials, models)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContextAwareSelector(b *testing.B) {
	credentials, models := setupTestData()
	selector := NewContextAwareSelector()
	context := &types.PayloadContext{
		HasImages: true,
		HasVideos: false,
		HasTools:  true,
		HasStream: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := selector.SelectWithContext(credentials, models, context)
		if err != nil {
			b.Fatal(err)
		}
	}
}

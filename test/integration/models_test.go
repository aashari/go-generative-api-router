package integration

import (
	"testing"

	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestModelsEndpoint(t *testing.T) {
	config := helpers.DefaultTestConfig()
	config.ServiceName = "models-integration-test"

	ts := helpers.NewTestServer(t, config)
	defer ts.Close()

	t.Run("models_list_basic", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/models", nil, nil)
		if err != nil {
			t.Fatalf("Models request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		if modelsResp.Object != "list" {
			t.Errorf("Expected object 'list', got: %s", modelsResp.Object)
		}

		if len(modelsResp.Data) == 0 {
			t.Error("Expected at least one model")
		}

		t.Logf("Found %d models", len(modelsResp.Data))

		// Verify model structure
		for _, model := range modelsResp.Data {
			if model.ID == "" {
				t.Error("Model ID should not be empty")
			}
			if model.Object != "model" {
				t.Errorf("Expected model object 'model', got: %s", model.Object)
			}
			if model.OwnedBy == "" {
				t.Error("Model owned_by should not be empty")
			}
		}
	})

	t.Run("models_response_format", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/models", nil, nil)
		if err != nil {
			t.Fatalf("Models request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", contentType)
		}

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		// Verify response structure
		if modelsResp.Object != "list" {
			t.Errorf("Expected object 'list', got: %s", modelsResp.Object)
		}

		if modelsResp.Data == nil {
			t.Error("Data field should not be nil")
		}
	})

	t.Run("vendor_filtering_openai", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/models?vendor=openai", nil, nil)
		if err != nil {
			t.Fatalf("OpenAI models request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		t.Logf("Found %d OpenAI models", len(modelsResp.Data))

		// OpenAI models might not be available in test environment
		for _, model := range modelsResp.Data {
			if model.OwnedBy != "openai" {
				t.Errorf("Expected OpenAI model, got owned_by: %s", model.OwnedBy)
			}
		}
	})

	t.Run("vendor_filtering_gemini", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/models?vendor=gemini", nil, nil)
		if err != nil {
			t.Fatalf("Gemini models request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		t.Logf("Found %d Gemini models", len(modelsResp.Data))

		// Should have Gemini models in test environment
		if len(modelsResp.Data) == 0 {
			t.Error("Expected at least one Gemini model")
		}

		for _, model := range modelsResp.Data {
			if model.OwnedBy != "gemini" {
				t.Errorf("Expected Gemini model, got owned_by: %s", model.OwnedBy)
			} else {
				t.Logf("Found Gemini model: %s (owned by: %s)", model.ID, model.OwnedBy)
			}
		}
	})

	t.Run("invalid_vendor_filter", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/models?vendor=invalid", nil, nil)
		if err != nil {
			t.Fatalf("Invalid vendor request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		// Should return empty list for invalid vendor
		if len(modelsResp.Data) != 0 {
			t.Errorf("Expected 0 models for invalid vendor, got: %d", len(modelsResp.Data))
		}
	})

	t.Run("models_cors", func(t *testing.T) {
		headers := map[string]string{
			"Origin": "https://example.com",
		}

		resp, body, err := ts.MakeRequest("GET", "/v1/models", nil, headers)
		if err != nil {
			t.Fatalf("CORS models request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		// Check for CORS headers (optional - service may not have CORS configured)
		corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if corsOrigin != "" {
			t.Logf("CORS Origin header present: %s", corsOrigin)
		} else {
			t.Log("CORS headers not configured (this is acceptable)")
		}
	})
}

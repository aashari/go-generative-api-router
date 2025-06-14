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
			t.Errorf("Expected object 'list', got %v", modelsResp.Object)
		}

		if len(modelsResp.Data) == 0 {
			t.Error("Expected at least one model")
		}

		t.Logf("Found %d models", len(modelsResp.Data))
	})

	t.Run("models_response_format", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/v1/models", nil, nil)
		if err != nil {
			t.Fatalf("Models request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var modelsResp helpers.ModelsResponse
		ts.AssertJSONResponse(body, &modelsResp)

		// Verify each model has required fields
		for i, model := range modelsResp.Data {
			if model.ID == "" {
				t.Errorf("Model %d missing ID", i)
			}
			if model.Object != "model" {
				t.Errorf("Model %d has wrong object type: %s", i, model.Object)
			}
			if model.OwnedBy == "" {
				t.Errorf("Model %d missing owned_by", i)
			}
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

		if modelsResp.Object != "list" {
			t.Errorf("Expected object 'list', got %v", modelsResp.Object)
		}

		t.Logf("Found %d OpenAI models", len(modelsResp.Data))

		// Verify all models are OpenAI models
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

		if modelsResp.Object != "list" {
			t.Errorf("Expected object 'list', got %v", modelsResp.Object)
		}

		t.Logf("Found %d Gemini models", len(modelsResp.Data))

		// Verify all models are Gemini models
		for _, model := range modelsResp.Data {
			if model.OwnedBy != "google" {
				t.Errorf("Expected Gemini model, got owned_by: %s", model.OwnedBy)
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
			t.Errorf("Expected empty list for invalid vendor, got %d models", len(modelsResp.Data))
		}
	})

	t.Run("models_cors", func(t *testing.T) {
		headers := map[string]string{
			"Origin": "https://example.com",
		}

		resp, _, err := ts.MakeRequest("GET", "/v1/models", nil, headers)
		if err != nil {
			t.Fatalf("Models CORS request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		// Check CORS headers
		if resp.Header.Get("Access-Control-Allow-Origin") == "" {
			t.Error("Missing CORS headers")
		}
	})
}

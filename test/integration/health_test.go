package integration

import (
	"testing"

	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestHealthEndpoint(t *testing.T) {
	config := helpers.DefaultTestConfig()
	config.ServiceName = "health-integration-test"

	ts := helpers.NewTestServer(t, config)
	defer ts.Close()

	t.Run("health_check_basic", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/health", nil, nil)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var healthResp helpers.HealthResponse
		ts.AssertJSONResponse(body, &healthResp)

		if healthResp.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got: %s", healthResp.Status)
		}

		t.Logf("Health check status: %s", healthResp.Status)
		t.Logf("Services: %v", healthResp.Services)

		// Verify essential services are present
		expectedServices := []string{"api", "credentials", "models", "selector"}
		for _, service := range expectedServices {
			if status, exists := healthResp.Services[service]; !exists {
				t.Errorf("Missing service: %s", service)
			} else if status != "up" {
				t.Errorf("Service %s is not up: %s", service, status)
			}
		}
	})

	t.Run("health_check_response_format", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/health", nil, nil)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		// Verify content type
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", contentType)
		}

		var healthResp helpers.HealthResponse
		ts.AssertJSONResponse(body, &healthResp)

		// Verify response structure
		if healthResp.Status == "" {
			t.Error("Status field should not be empty")
		}

		if healthResp.Services == nil {
			t.Error("Services field should not be nil")
		}
	})

	t.Run("health_check_cors", func(t *testing.T) {
		headers := map[string]string{
			"Origin": "https://example.com",
		}

		resp, body, err := ts.MakeRequest("GET", "/health", nil, headers)
		if err != nil {
			t.Fatalf("Health CORS request failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var healthResp helpers.HealthResponse
		ts.AssertJSONResponse(body, &healthResp)

		// Check for CORS headers (optional - service may not have CORS configured)
		corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if corsOrigin != "" {
			t.Logf("CORS Origin header present: %s", corsOrigin)
		} else {
			t.Log("CORS headers not configured (this is acceptable)")
		}

		// Health endpoint should still work regardless of CORS configuration
		if healthResp.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got: %s", healthResp.Status)
		}
	})
}

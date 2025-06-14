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

		if healthResp.Status == "" {
			t.Error("Health response missing status")
		}

		if healthResp.Services == nil {
			t.Error("Health response missing services")
		}

		t.Logf("Health check status: %v", healthResp.Status)
		t.Logf("Services: %v", healthResp.Services)
	})

	t.Run("health_check_response_format", func(t *testing.T) {
		resp, body, err := ts.MakeRequest("GET", "/health", nil, nil)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		var healthResp helpers.HealthResponse
		ts.AssertJSONResponse(body, &healthResp)

		// Verify required fields
		if healthResp.Status == "" {
			t.Error("Missing status field")
		}

		if healthResp.Timestamp == "" {
			t.Error("Missing timestamp field")
		}

		if healthResp.Services == nil {
			t.Error("Missing services field")
		}

		// Verify expected services
		expectedServices := []string{"api", "credentials", "models", "selector"}
		for _, service := range expectedServices {
			if _, exists := healthResp.Services[service]; !exists {
				t.Errorf("Missing service: %s", service)
			}
		}
	})

	t.Run("health_check_cors", func(t *testing.T) {
		headers := map[string]string{
			"Origin": "https://example.com",
		}

		resp, _, err := ts.MakeRequest("GET", "/health", nil, headers)
		if err != nil {
			t.Fatalf("Health check with CORS failed: %v", err)
		}

		ts.AssertStatusCode(resp, 200)

		// Check CORS headers
		if resp.Header.Get("Access-Control-Allow-Origin") == "" {
			t.Error("Missing CORS headers")
		}
	})
}

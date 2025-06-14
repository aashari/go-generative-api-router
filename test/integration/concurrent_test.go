package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aashari/go-generative-api-router/test/helpers"
)

func TestConcurrentRequests(t *testing.T) {
	ts := helpers.SetupTestServer(t)
	defer ts.Teardown()

	t.Run("handle_concurrent_requests", func(t *testing.T) {
		concurrentRequests := 10
		var wg sync.WaitGroup
		wg.Add(concurrentRequests)

		successCount := 0
		var mu sync.Mutex

		for i := 0; i < concurrentRequests; i++ {
			go func(index int) {
				defer wg.Done()

				request := helpers.ChatCompletionRequest{
					Model: "gpt-4o",
					Messages: []helpers.Message{
						{
							Role:    "user",
							Content: fmt.Sprintf("Concurrent request #%d", index+1),
						},
					},
					MaxTokens: 10,
				}

				resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
				if err != nil {
					t.Logf("Concurrent request %d failed: %v", index+1, err)
					return
				}

				if resp.StatusCode == 200 {
					mu.Lock()
					successCount++
					mu.Unlock()
				} else {
					t.Logf("Concurrent request %d returned status %d: %s", index+1, resp.StatusCode, string(body))
				}
			}(i)
		}

		wg.Wait()
		t.Logf("Concurrent requests completed: %d/%d successful", successCount, concurrentRequests)

		// At least some requests should succeed
		if successCount == 0 {
			t.Error("All concurrent requests failed")
		}
	})

	t.Run("rapid_successive_requests", func(t *testing.T) {
		// Test rapid requests to verify load balancing/distribution
		successCount := 0
		requestCount := 20

		for i := 0; i < requestCount; i++ {
			request := helpers.ChatCompletionRequest{
				Model: "gpt-4o",
				Messages: []helpers.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("Rapid test #%d", i+1),
					},
				},
				MaxTokens: 5,
			}

			resp, body, err := ts.MakeRequest("POST", "/v1/chat/completions", request, nil)
			if err != nil {
				t.Logf("Rapid request %d failed: %v", i+1, err)
				continue
			}

			if resp.StatusCode == 200 {
				successCount++
			} else {
				t.Logf("Rapid request %d returned status %d: %s", i+1, resp.StatusCode, string(body))
			}

			// Small delay to avoid overwhelming APIs
			time.Sleep(100 * time.Millisecond)
		}

		t.Logf("Rapid requests test: %d/%d successful", successCount, requestCount)
	})
} 
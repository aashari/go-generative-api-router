package proxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test image data (1x1 transparent PNG)
const testImageBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func TestImageProcessor_ProcessMessageContent(t *testing.T) {
	processor := NewImageProcessor()
	ctx := context.Background()

	tests := []struct {
		name     string
		content  interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "string content unchanged",
			content:  "Hello, world!",
			expected: "Hello, world!",
			wantErr:  false,
		},
		{
			name: "array with text only",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "What's in this image?",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "What's in this image?",
				},
			},
			wantErr: false,
		},
		{
			name: "array with base64 image unchanged",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Describe this",
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "data:image/png;base64," + testImageBase64,
					},
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Describe this",
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "data:image/png;base64," + testImageBase64,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessMessageContent(ctx, tt.content)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestImageProcessor_isPublicURL(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		url      string
		expected bool
	}{
		{"http://example.com/image.jpg", true},
		{"https://example.com/image.png", true},
		{"HTTP://EXAMPLE.COM", false}, // Case sensitive
		{"data:image/png;base64,iVBORw0KG", false},
		{"file:///path/to/image.jpg", false},
		{"ftp://example.com/image.jpg", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := processor.isPublicURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageProcessor_isValidImageType(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		contentType string
		expected    bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/jpg", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/png; charset=utf-8", true}, // With parameters
		{"text/html", false},
		{"application/json", false},
		{"image/svg+xml", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := processor.isValidImageType(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageProcessor_isValidImageType_Enhanced(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		contentType string
		expected    bool
		description string
	}{
		// Explicit image types
		{"image/png", true, "PNG image"},
		{"image/jpeg", true, "JPEG image"},
		{"image/jpg", true, "JPG image"},
		{"image/gif", true, "GIF image"},
		{"image/webp", true, "WebP image"},
		{"image/bmp", true, "BMP image"},
		{"image/tiff", true, "TIFF image"},
		{"image/svg+xml", true, "SVG image"},
		{"image/png; charset=utf-8", true, "PNG with parameters"},

		// Generic types that might contain images
		{"application/octet-stream", true, "Generic binary (Telegram, Discord)"},
		{"binary/octet-stream", true, "Binary octet stream"},
		{"application/binary", true, "Application binary"},

		// Invalid types
		{"text/html", false, "HTML content"},
		{"application/json", false, "JSON content"},
		{"video/mp4", false, "Video content"},
		{"", false, "Empty content type"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := processor.isValidImageType(tt.contentType)
			assert.Equal(t, tt.expected, result, "Content type: %s", tt.contentType)
		})
	}
}

func TestImageProcessor_detectImageFormat(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		name         string
		data         []byte
		expectedType string
		expectedOK   bool
	}{
		{
			name:         "PNG magic number",
			data:         []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
			expectedType: "image/png",
			expectedOK:   true,
		},
		{
			name:         "JPEG magic number",
			data:         []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01},
			expectedType: "image/jpeg",
			expectedOK:   true,
		},
		{
			name:         "GIF magic number",
			data:         []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00},
			expectedType: "image/gif",
			expectedOK:   true,
		},
		{
			name:         "WebP magic number",
			data:         []byte{0x52, 0x49, 0x46, 0x46, 0x24, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			expectedType: "image/webp",
			expectedOK:   true,
		},
		{
			name:         "BMP magic number",
			data:         []byte{0x42, 0x4D, 0x36, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0x00},
			expectedType: "image/bmp",
			expectedOK:   true,
		},
		{
			name:         "TIFF little endian",
			data:         []byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedType: "image/tiff",
			expectedOK:   true,
		},
		{
			name:         "TIFF big endian",
			data:         []byte{0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00},
			expectedType: "image/tiff",
			expectedOK:   true,
		},
		{
			name:         "Not an image",
			data:         []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00}, // ZIP
			expectedType: "",
			expectedOK:   false,
		},
		{
			name:         "Too short data",
			data:         []byte{0x89, 0x50},
			expectedType: "",
			expectedOK:   false,
		},
		{
			name:         "Empty data",
			data:         []byte{},
			expectedType: "",
			expectedOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detectedType, ok := processor.detectImageFormat(tt.data)
			assert.Equal(t, tt.expectedOK, ok, "Detection should match expected result")
			if tt.expectedOK {
				assert.Equal(t, tt.expectedType, detectedType, "Detected type should match expected")
			}
		})
	}
}

func TestImageProcessor_downloadAndConvertImage(t *testing.T) {
	// Create test image server
	imageData, _ := base64.StdEncoding.DecodeString(testImageBase64)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/valid.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(imageData)
		case "/invalid-type":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html>Not an image</html>"))
		case "/not-found":
			w.WriteHeader(http.StatusNotFound)
		case "/large-image":
			w.Header().Set("Content-Type", "image/png")
			// Write more than 20MB
			for i := 0; i < 21*1024*1024; i++ {
				w.Write([]byte{0})
			}
		case "/slow-image":
			w.Header().Set("Content-Type", "image/png")
			time.Sleep(40 * time.Second) // Longer than timeout
			w.Write(imageData)
		}
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid image download",
			url:     server.URL + "/valid.png",
			wantErr: false,
		},
		{
			name:    "invalid content type",
			url:     server.URL + "/invalid-type",
			wantErr: true,
			errMsg:  "invalid content type",
		},
		{
			name:    "404 not found",
			url:     server.URL + "/not-found",
			wantErr: true,
			errMsg:  "status 404",
		},
		{
			name:    "image too large",
			url:     server.URL + "/large-image",
			wantErr: true,
			errMsg:  "exceeds limit",
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
			errMsg:  "failed to download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.downloadAndConvertImage(ctx, tt.url)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, strings.HasPrefix(result, "data:image/"))
				assert.Contains(t, result, ";base64,")
			}
		})
	}
}

func TestImageProcessor_ProcessContentParts_Concurrent(t *testing.T) {
	// Create test server with multiple images
	imageData, _ := base64.StdEncoding.DecodeString(testImageBase64)
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	// Create multiple image URLs
	parts := []ContentPart{
		{Type: "text", Text: "Compare these images:"},
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/image1.png"}},
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/image2.png"}},
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/image3.png"}},
		{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/png;base64," + testImageBase64}}, // Already base64
	}

	// Process concurrently
	result, err := processor.processContentParts(ctx, parts)

	require.NoError(t, err)
	assert.Len(t, result, len(parts))

	// Check text part unchanged
	assert.Equal(t, "text", result[0].Type)
	assert.Equal(t, "Compare these images:", result[0].Text)

	// Check public URLs were converted
	for i := 1; i <= 3; i++ {
		assert.Equal(t, "image_url", result[i].Type)
		assert.True(t, strings.HasPrefix(result[i].ImageURL.URL, "data:image/png;base64,"))
	}

	// Check base64 URL unchanged
	assert.Equal(t, "data:image/png;base64,"+testImageBase64, result[4].ImageURL.URL)

	// Verify concurrent processing happened (3 public URLs)
	assert.Equal(t, 3, requestCount)
}

func TestImageProcessor_ProcessRequestBody(t *testing.T) {
	// Create test server
	imageData, _ := base64.StdEncoding.DecodeString(testImageBase64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	tests := []struct {
		name         string
		body         string
		shouldModify bool
		wantErr      bool
	}{
		{
			name: "simple text message",
			body: `{
				"model": "gpt-4",
				"messages": [{"role": "user", "content": "Hello"}]
			}`,
			shouldModify: false,
			wantErr:      false,
		},
		{
			name: "vision message with public URL",
			body: fmt.Sprintf(`{
				"model": "vision-model",
				"messages": [{
					"role": "user",
					"content": [
						{"type": "text", "text": "What's this?"},
						{"type": "image_url", "image_url": {"url": "%s/test.png"}}
					]
				}]
			}`, server.URL),
			shouldModify: true,
			wantErr:      false,
		},
		{
			name: "vision message with base64",
			body: `{
				"model": "vision-model",
				"messages": [{
					"role": "user",
					"content": [
						{"type": "text", "text": "What's this?"},
						{"type": "image_url", "image_url": {"url": "data:image/png;base64,` + testImageBase64 + `"}}
					]
				}]
			}`,
			shouldModify: false,
			wantErr:      false,
		},
		{
			name: "multiple messages with mixed content",
			body: fmt.Sprintf(`{
				"model": "vision-model",
				"messages": [
					{"role": "system", "content": "You are helpful"},
					{"role": "user", "content": "First question"},
					{"role": "assistant", "content": "First answer"},
					{"role": "user", "content": [
						{"type": "text", "text": "What's this?"},
						{"type": "image_url", "image_url": {"url": "%s/test.png"}}
					]}
				]
			}`, server.URL),
			shouldModify: true,
			wantErr:      false,
		},
		{
			name:    "invalid JSON",
			body:    `{invalid json}`,
			wantErr: true,
		},
		{
			name:         "no messages field",
			body:         `{"model": "gpt-4"}`,
			shouldModify: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessRequestBody(ctx, []byte(tt.body))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.shouldModify {
					// Check that result is different from input
					assert.NotEqual(t, tt.body, string(result))

					// Verify result is valid JSON
					var parsed map[string]interface{}
					err := json.Unmarshal(result, &parsed)
					assert.NoError(t, err)

					// Verify URLs were converted to base64
					if messages, ok := parsed["messages"].([]interface{}); ok {
						for _, msg := range messages {
							if msgMap, ok := msg.(map[string]interface{}); ok {
								if content, ok := msgMap["content"].([]interface{}); ok {
									for _, part := range content {
										if partMap, ok := part.(map[string]interface{}); ok {
											if partMap["type"] == "image_url" {
												if imgURL, ok := partMap["image_url"].(map[string]interface{}); ok {
													url := imgURL["url"].(string)
													if !strings.HasPrefix(url, "data:") {
														t.Errorf("Expected base64 URL, got: %s", url)
													}
												}
											}
										}
									}
								}
							}
						}
					}
				} else {
					// Result should be unchanged
					assert.JSONEq(t, tt.body, string(result))
				}
			}
		})
	}
}

func TestImageProcessor_ProcessRequestBodyWithHeaders(t *testing.T) {
	// Create test server that requires authentication
	authToken := "Bearer test-token-123"
	imageData, _ := base64.StdEncoding.DecodeString(testImageBase64)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the authorization header is present
		if r.Header.Get("Authorization") != authToken {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		// Check custom header
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing custom header"))
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(imageData)
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	// Test request with headers
	requestBody := fmt.Sprintf(`{
		"model": "vision-model",
		"messages": [{
			"role": "user",
			"content": [
				{"type": "text", "text": "What's this?"},
				{
					"type": "image_url", 
					"image_url": {
						"url": "%s/test.png",
						"headers": {
							"Authorization": "%s",
							"X-Custom-Header": "custom-value"
						}
					}
				}
			]
		}]
	}`, server.URL, authToken)

	processedBody, err := processor.ProcessRequestBody(ctx, []byte(requestBody))
	assert.NoError(t, err)

	// Parse the processed body
	var processedData map[string]interface{}
	err = json.Unmarshal(processedBody, &processedData)
	assert.NoError(t, err)

	// Verify the structure
	messages := processedData["messages"].([]interface{})
	message := messages[0].(map[string]interface{})
	content := message["content"].([]interface{})

	// Find the image_url part
	var imageURLPart map[string]interface{}
	for _, part := range content {
		if partMap := part.(map[string]interface{}); partMap["type"] == "image_url" {
			imageURLPart = partMap
			break
		}
	}

	assert.NotNil(t, imageURLPart)
	imageURL := imageURLPart["image_url"].(map[string]interface{})

	// Verify URL was converted to base64
	url := imageURL["url"].(string)
	assert.True(t, strings.HasPrefix(url, "data:image/png;base64,"))

	// Verify headers are NOT present in the final output (removed for vendor compatibility)
	_, headersExist := imageURL["headers"]
	assert.False(t, headersExist, "Headers should be removed from the final output")
}

func TestImageProcessor_ProcessRequestBodyWithMissingHeaders(t *testing.T) {
	// Create test server that requires authentication
	authToken := "Bearer test-token-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the authorization header is present
		if r.Header.Get("Authorization") != authToken {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	// Test request without required headers (should now succeed with graceful error handling)
	requestBody := fmt.Sprintf(`{
		"model": "vision-model",
		"messages": [{
			"role": "user",
			"content": [
				{"type": "text", "text": "What's this?"},
				{
					"type": "image_url", 
					"image_url": {
						"url": "%s/test.png"
					}
				}
			]
		}]
	}`, server.URL)

	processedBody, err := processor.ProcessRequestBody(ctx, []byte(requestBody))
	assert.NoError(t, err) // Should succeed with graceful error handling

	// Parse the processed body to verify the failed image was replaced with system message
	var processedData map[string]interface{}
	err = json.Unmarshal(processedBody, &processedData)
	assert.NoError(t, err)

	// Verify the structure
	messages := processedData["messages"].([]interface{})
	message := messages[0].(map[string]interface{})
	content := message["content"].([]interface{})

	// Find the text part that should contain the system message
	var systemMessagePart map[string]interface{}
	for _, part := range content {
		if partMap := part.(map[string]interface{}); partMap["type"] == "text" {
			if text, ok := partMap["text"].(string); ok && strings.Contains(text, "<system>") {
				systemMessagePart = partMap
				break
			}
		}
	}

	assert.NotNil(t, systemMessagePart, "Should have a system message for failed image")
	systemText := systemMessagePart["text"].(string)
	assert.Contains(t, systemText, "authentication or access permissions")
	assert.Contains(t, systemText, "<system>")
}

func TestImageProcessor_ConcurrentErrorHandling(t *testing.T) {
	// Create test server that returns errors for some images
	successCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success1.png", "/success2.png":
			imageData, _ := base64.StdEncoding.DecodeString(testImageBase64)
			w.Header().Set("Content-Type", "image/png")
			w.Write(imageData)
			successCount++
		case "/error1.png":
			w.WriteHeader(http.StatusInternalServerError)
		case "/error2.png":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("Not an image"))
		}
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	parts := []ContentPart{
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/success1.png"}},
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/error1.png"}},
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/success2.png"}},
		{Type: "image_url", ImageURL: &ImageURL{URL: server.URL + "/error2.png"}},
	}

	// Process should now succeed with graceful error handling
	result, err := processor.processContentParts(ctx, parts)

	assert.NoError(t, err) // Should succeed with graceful error handling
	assert.Len(t, result, len(parts))

	// Verify successful images were processed
	assert.Equal(t, "image_url", result[0].Type)
	assert.True(t, strings.HasPrefix(result[0].ImageURL.URL, "data:image/png;base64,"))

	assert.Equal(t, "image_url", result[2].Type)
	assert.True(t, strings.HasPrefix(result[2].ImageURL.URL, "data:image/png;base64,"))

	// Verify failed images were replaced with system messages
	assert.Equal(t, "text", result[1].Type)
	assert.Contains(t, result[1].Text, "<system>")
	assert.Contains(t, result[1].Text, "technical issue")

	assert.Equal(t, "text", result[3].Type)
	assert.Contains(t, result[3].Text, "<system>")
	assert.Contains(t, result[3].Text, "not an image format")
}

func TestImageProcessor_downloadWithGenericContentType(t *testing.T) {
	// Create test image data (1x1 PNG)
	pngData, _ := base64.StdEncoding.DecodeString(testImageBase64)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/telegram-style.jpg":
			// Simulate Telegram-style response with generic content type but actual image data
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(pngData) // PNG data with generic content type
		case "/discord-style.png":
			// Simulate Discord-style response
			w.Header().Set("Content-Type", "binary/octet-stream")
			w.Write(pngData)
		case "/generic-not-image":
			// Generic content type but not actually an image
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("This is not image data"))
		case "/proper-image.png":
			// Proper content type
			w.Header().Set("Content-Type", "image/png")
			w.Write(pngData)
		}
	}))
	defer server.Close()

	processor := NewImageProcessor()
	ctx := context.Background()

	tests := []struct {
		name        string
		url         string
		wantErr     bool
		errContains string
		expectType  string
	}{
		{
			name:       "Telegram-style generic content type with PNG data",
			url:        server.URL + "/telegram-style.jpg",
			wantErr:    false,
			expectType: "image/png", // Should detect PNG from magic numbers
		},
		{
			name:       "Discord-style binary content type with PNG data",
			url:        server.URL + "/discord-style.png",
			wantErr:    false,
			expectType: "image/png",
		},
		{
			name:        "Generic content type but not image data",
			url:         server.URL + "/generic-not-image",
			wantErr:     true,
			errContains: "not a valid image format",
		},
		{
			name:       "Proper image content type",
			url:        server.URL + "/proper-image.png",
			wantErr:    false,
			expectType: "image/png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.downloadAndConvertImage(ctx, tt.url)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, strings.HasPrefix(result, "data:"+tt.expectType+";base64,"))
				assert.Contains(t, result, ";base64,")
			}
		})
	}
}

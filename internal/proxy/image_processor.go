package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// ImageProcessor handles image URL processing and conversion
type ImageProcessor struct {
	httpClient *http.Client
	maxSize    int64
}

// NewImageProcessor creates a new image processor with default settings
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxSize: 20 * 1024 * 1024, // 20MB limit
	}
}

// ContentPart represents a part of the message content
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image URL structure
type ImageURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ProcessResult holds the result of processing a content part
type ProcessResult struct {
	Index   int
	Content ContentPart
	Error   error
}

// ProcessMessageContent processes message content and converts public URLs to base64
func (p *ImageProcessor) ProcessMessageContent(ctx context.Context, content interface{}) (interface{}, error) {
	// Handle string content (backwards compatibility)
	if str, ok := content.(string); ok {
		return str, nil
	}

	// Handle array content (vision requests)
	if arr, ok := content.([]interface{}); ok {
		return p.processContentArray(ctx, arr)
	}

	// If it's already a structured content array, process it
	if parts, ok := content.([]ContentPart); ok {
		return p.processContentParts(ctx, parts)
	}

	return content, nil
}

// processContentArray processes an array of content parts
func (p *ImageProcessor) processContentArray(ctx context.Context, arr []interface{}) ([]interface{}, error) {
	// First, convert to ContentPart structs for easier processing
	parts := make([]ContentPart, 0, len(arr))
	for _, item := range arr {
		if itemMap, ok := item.(map[string]interface{}); ok {
			part := ContentPart{}

			// Extract type
			if typeVal, ok := itemMap["type"].(string); ok {
				part.Type = typeVal
			}

			// Extract text
			if textVal, ok := itemMap["text"].(string); ok {
				part.Text = textVal
			}

			// Extract image_url
			if imageURLVal, ok := itemMap["image_url"].(map[string]interface{}); ok {
				imageURL := &ImageURL{}

				// Extract URL
				if urlStr, ok := imageURLVal["url"].(string); ok {
					imageURL.URL = urlStr
				}

				// Extract headers if present
				if headersVal, ok := imageURLVal["headers"].(map[string]interface{}); ok {
					headers := make(map[string]string)
					for key, value := range headersVal {
						if strValue, ok := value.(string); ok {
							headers[key] = strValue
						}
					}
					imageURL.Headers = headers
				}

				part.ImageURL = imageURL
			}

			parts = append(parts, part)
		}
	}

	// Process the parts
	processedParts, err := p.processContentParts(ctx, parts)
	if err != nil {
		return nil, err
	}

	// Convert back to interface array
	result := make([]interface{}, len(processedParts))
	for i, part := range processedParts {
		partMap := map[string]interface{}{
			"type": part.Type,
		}

		if part.Type == "text" && part.Text != "" {
			partMap["text"] = part.Text
		}

		if part.Type == "image_url" && part.ImageURL != nil {
			// Create image_url object without headers (headers are removed for vendor compatibility)
			imageURLMap := map[string]interface{}{
				"url": part.ImageURL.URL,
			}
			partMap["image_url"] = imageURLMap
		}

		result[i] = partMap
	}

	return result, nil
}

// processContentParts processes content parts concurrently
func (p *ImageProcessor) processContentParts(ctx context.Context, parts []ContentPart) ([]ContentPart, error) {
	// Find all image URLs that need processing
	imagesToProcess := make(map[int]int) // maps result index to parts index
	resultIndex := 0
	for i, part := range parts {
		if part.Type == "image_url" && part.ImageURL != nil && p.isPublicURL(part.ImageURL.URL) {
			imagesToProcess[resultIndex] = i
			resultIndex++
		}
	}

	if len(imagesToProcess) == 0 {
		// No images to process
		return parts, nil
	}

	// Log image processing start
	logger.LogMultipleData(ctx, logger.LevelInfo, "Processing image URLs concurrently", map[string]any{
		"image_count":       len(imagesToProcess),
		"total_parts":       len(parts),
		"images_to_process": imagesToProcess,
	})

	// Process images concurrently
	results := make(chan ProcessResult, len(imagesToProcess))
	var wg sync.WaitGroup
	wg.Add(len(imagesToProcess))

	for resultIdx, partIdx := range imagesToProcess {
		go func(rIdx, pIdx int) {
			defer wg.Done()

			part := parts[pIdx]
			processedURL, err := p.downloadAndConvertImageWithHeaders(ctx, part.ImageURL.URL, part.ImageURL.Headers)

			result := ProcessResult{
				Index: pIdx,
				Content: ContentPart{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: processedURL,
						// Note: Headers are intentionally omitted here to remove them from vendor request
					},
				},
				Error: err,
			}

			results <- result
		}(resultIdx, partIdx)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	processedParts := make([]ContentPart, len(parts))
	copy(processedParts, parts)

	var errors []error
	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("image at index %d: %w", result.Index, result.Error))
		} else {
			processedParts[result.Index] = result.Content
		}
	}

	// Log processing completion
	logger.LogMultipleData(ctx, logger.LevelInfo, "Image URL processing completed", map[string]any{
		"processed_count": len(imagesToProcess),
		"error_count":     len(errors),
		"errors":          errors,
	})

	// If any errors occurred, return the first one
	if len(errors) > 0 {
		return nil, fmt.Errorf("failed to process %d images: %w", len(errors), errors[0])
	}

	return processedParts, nil
}

// isPublicURL checks if a URL is a public HTTP/HTTPS URL
func (p *ImageProcessor) isPublicURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

// downloadAndConvertImage downloads an image from a URL and converts it to base64 (backward compatibility)
func (p *ImageProcessor) downloadAndConvertImage(ctx context.Context, imageURL string) (string, error) {
	return p.downloadAndConvertImageWithHeaders(ctx, imageURL, nil)
}

// downloadAndConvertImageWithHeaders downloads an image from a URL with custom headers and converts it to base64
func (p *ImageProcessor) downloadAndConvertImageWithHeaders(ctx context.Context, imageURL string, headers map[string]string) (string, error) {
	logger.LogMultipleData(ctx, logger.LevelDebug, "Downloading image from URL with headers", map[string]any{
		"url":     imageURL,
		"headers": headers,
	})

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocks
	req.Header.Set("User-Agent", "Generative-API-Router/1.0")

	// Add custom headers if provided
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
			logger.LogMultipleData(ctx, logger.LevelDebug, "Added custom header for image download", map[string]any{
				"header_key":   key,
				"header_value": value,
				"url":          imageURL,
			})
		}
	}

	// Download the image
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !p.isValidImageType(contentType) {
		return "", fmt.Errorf("invalid content type: %s", contentType)
	}

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, p.maxSize)
	imageData, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(imageData)) >= p.maxSize {
		return "", fmt.Errorf("image size exceeds limit of %d bytes", p.maxSize)
	}

	// Convert to base64 with data URL scheme
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)

	logger.LogMultipleData(ctx, logger.LevelDebug, "Image downloaded and converted", map[string]any{
		"original_url":  imageURL,
		"content_type":  contentType,
		"size_bytes":    len(imageData),
		"base64_length": len(base64Data),
		"data_url":      dataURL, // This will be automatically truncated by the logger
	})

	return dataURL, nil
}

// isValidImageType checks if the content type is a supported image format
func (p *ImageProcessor) isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/png",
		"image/jpeg",
		"image/jpg",
		"image/gif",
		"image/webp",
	}

	for _, validType := range validTypes {
		if strings.HasPrefix(contentType, validType) {
			return true
		}
	}
	return false
}

// ProcessRequestBody processes the entire request body to handle image URLs
func (p *ImageProcessor) ProcessRequestBody(ctx context.Context, body []byte) ([]byte, error) {
	// Parse the request body
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return nil, fmt.Errorf("invalid request format: %v", err)
	}

	// Check if messages exist
	messages, ok := requestData["messages"].([]interface{})
	if !ok {
		// No messages or wrong format, return as-is
		return body, nil
	}

	// Process each message
	modified := false
	for i, msg := range messages {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if content, exists := msgMap["content"]; exists {
				processedContent, err := p.ProcessMessageContent(ctx, content)
				if err != nil {
					return nil, fmt.Errorf("failed to process message %d: %w", i, err)
				}

				// Check if content was modified
				if !bytes.Equal(mustMarshal(content), mustMarshal(processedContent)) {
					msgMap["content"] = processedContent
					messages[i] = msgMap
					modified = true
				}
			}
		}
	}

	// If nothing was modified, return original body
	if !modified {
		return body, nil
	}

	// Re-encode the modified request
	requestData["messages"] = messages
	modifiedBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode modified request: %v", err)
	}

	return modifiedBody, nil
}

// mustMarshal is a helper to marshal for comparison
func mustMarshal(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

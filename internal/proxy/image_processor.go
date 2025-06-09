package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

// processContentParts processes content parts concurrently with graceful error handling
func (p *ImageProcessor) processContentParts(ctx context.Context, parts []ContentPart) ([]ContentPart, error) {
	// Find all image URLs and files that need processing
	itemsToProcess := make(map[int]int) // maps result index to parts index
	resultIndex := 0
	for i, part := range parts {
		if (part.Type == "image_url" || part.Type == "file") && part.ImageURL != nil && p.isPublicURL(part.ImageURL.URL) {
			itemsToProcess[resultIndex] = i
			resultIndex++
		}
	}

	if len(itemsToProcess) == 0 {
		// No items to process
		return parts, nil
	}

	// Count total items in the request (including non-public URLs)
	totalItems := 0
	for _, part := range parts {
		if (part.Type == "image_url" || part.Type == "file") && part.ImageURL != nil {
			totalItems++
		}
	}

	// Log processing start
	logger.LogWithStructure(ctx, logger.LevelInfo, "Processing image URLs and files concurrently",
		map[string]interface{}{
			"item_count":       len(itemsToProcess),
			"total_parts":      len(parts),
			"total_items":      totalItems,
			"items_to_process": itemsToProcess,
		},
		nil, // request
		nil, // response
		nil) // error

	// Process items concurrently
	results := make(chan ProcessResult, len(itemsToProcess))
	var wg sync.WaitGroup
	wg.Add(len(itemsToProcess))

	for resultIdx, partIdx := range itemsToProcess {
		go func(rIdx, pIdx int) {
			defer wg.Done()

			part := parts[pIdx]
			var processedContent ContentPart
			var err error

			if part.Type == "image_url" {
				// Process image
				processedURL, imgErr := p.downloadAndConvertImageWithHeaders(ctx, part.ImageURL.URL, part.ImageURL.Headers)
				err = imgErr
				processedContent = ContentPart{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: processedURL,
						// Note: Headers are intentionally omitted here to remove them from vendor request
					},
				}
			} else if part.Type == "file" {
				// Process file
				fileText, fileErr := p.downloadAndConvertFileWithHeaders(ctx, part.ImageURL.URL, part.ImageURL.Headers)
				err = fileErr
				processedContent = ContentPart{
					Type: "text",
					Text: fileText,
				}
			}

			result := ProcessResult{
				Index:   pIdx,
				Content: processedContent,
				Error:   err,
			}

			results <- result
		}(resultIdx, partIdx)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with graceful error handling
	processedParts := make([]ContentPart, len(parts))
	copy(processedParts, parts)

	var errors []error
	var failedItems []int
	for result := range results {
		if result.Error != nil {
			// Instead of failing the entire request, replace failed item with system message
			itemType := parts[result.Index].Type
			errors = append(errors, fmt.Errorf("%s at index %d: %w", itemType, result.Index, result.Error))
			failedItems = append(failedItems, result.Index)

			// Calculate item position for better context
			itemPosition := 1
			for i := 0; i <= result.Index; i++ {
				if (parts[i].Type == "image_url" || parts[i].Type == "file") && parts[i].ImageURL != nil {
					if i == result.Index {
						break
					}
					itemPosition++
				}
			}

			// Generate contextual system message for failed item
			var systemMessage string
			if itemType == "file" {
				systemMessage = p.generateFileFailureSystemMessage(result.Error, itemPosition, totalItems, len(itemsToProcess) > 1)
			} else {
				systemMessage = p.generateImageFailureSystemMessage(result.Error, itemPosition, totalItems, len(itemsToProcess) > 1)
			}
			processedParts[result.Index] = ContentPart{
				Type: "text",
				Text: systemMessage,
			}

			logger.LogWithStructure(ctx, logger.LevelWarn, "Item processing failed, replaced with system message",
				map[string]interface{}{
					"item_type":      itemType,
					"item_index":     result.Index,
					"item_position":  itemPosition,
					"total_items":    totalItems,
					"mixed_scenario": len(itemsToProcess) > 1,
					"error":          result.Error.Error(),
					"system_message": systemMessage,
				},
				nil, // request
				nil, // response
				nil) // error
		} else {
			processedParts[result.Index] = result.Content
		}
	}

	// Log processing completion with graceful handling summary
	logger.LogWithStructure(ctx, logger.LevelInfo, "Item processing completed with graceful error handling",
		map[string]interface{}{
			"processed_count":     len(itemsToProcess),
			"successful_count":    len(itemsToProcess) - len(errors),
			"failed_count":        len(errors),
			"failed_item_indices": failedItems,
			"total_items":         totalItems,
			"mixed_scenario":      len(itemsToProcess) > 1 && len(errors) > 0 && len(errors) < len(itemsToProcess),
			"errors":              errors,
			"graceful_handling":   len(errors) > 0,
		},
		nil, // request
		nil, // response
		nil) // error

	// Always return success - errors are now handled gracefully
	return processedParts, nil
}

// generateImageFailureSystemMessage creates a contextual system message for failed image downloads
func (p *ImageProcessor) generateImageFailureSystemMessage(err error, imagePosition, totalImages int, hasMixedScenario bool) string {
	// Determine the type of error for more specific messaging
	errorMsg := err.Error()
	var baseMessage string
	var contextPrefix string

	// Create context prefix for mixed scenarios
	if hasMixedScenario && totalImages > 1 {
		contextPrefix = fmt.Sprintf("Image %d of %d could not be processed. ", imagePosition, totalImages)
	} else if totalImages > 1 {
		contextPrefix = fmt.Sprintf("One of the %d images provided could not be processed. ", totalImages)
	} else {
		contextPrefix = "The image provided could not be processed. "
	}

	// Determine specific error message based on error type
	if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
		baseMessage = "Respond naturally that you couldn't access the image due to network connectivity issues. The image server appears to be unreachable or the domain doesn't exist. Ask the user to verify the URL or provide an alternative image."
	} else if strings.Contains(errorMsg, "status 401") || strings.Contains(errorMsg, "status 403") {
		baseMessage = "Respond naturally that the image requires authentication or access permissions that weren't provided. The image couldn't be accessed due to authorization issues. Suggest they provide proper authentication headers or use a publicly accessible image."
	} else if strings.Contains(errorMsg, "status 404") {
		baseMessage = "Respond naturally that the image URL appears to be broken or the image has been moved/deleted (404 Not Found). Ask them to provide a valid image URL."
	} else if strings.Contains(errorMsg, "invalid content type") {
		baseMessage = "Respond naturally that the URL doesn't point to a valid image file. The content isn't an image format that can be processed. Ask them to provide a direct link to an image file (PNG, JPEG, GIF, WebP, etc.)."
	} else if strings.Contains(errorMsg, "size exceeds limit") {
		baseMessage = "Respond naturally that the image file is too large to process (exceeds 20MB limit). Ask them to provide a smaller image or compress it before sharing."
	} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
		baseMessage = "Respond naturally that the image took too long to download due to slow response from the image server. Suggest they try again later or provide an alternative image."
	} else {
		// Generic error message for unknown error types
		baseMessage = "Respond naturally that there was a technical issue processing this image. Ask them to try providing the image again or use an alternative image."
	}

	// Add guidance for mixed scenarios
	var mixedScenarioGuidance string
	if hasMixedScenario && totalImages > 1 {
		mixedScenarioGuidance = " You can still analyze and respond to the other images that were successfully processed."
	}

	// Construct the complete system message
	systemMessage := fmt.Sprintf("<system>\n%s%s%s The user cannot see this system message, so respond naturally as part of the ongoing conversation.\n</system>",
		contextPrefix, baseMessage, mixedScenarioGuidance)

	return systemMessage
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
	logger.LogWithStructure(ctx, logger.LevelDebug, "Downloading image from URL with headers",
		map[string]interface{}{
			"url":     imageURL,
			"headers": headers,
		},
		nil, // request
		nil, // response
		nil) // error

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
			logger.LogWithStructure(ctx, logger.LevelDebug, "Added custom header for image download",
				map[string]interface{}{
					"header_key":   key,
					"header_value": value,
					"url":          imageURL,
				},
				nil, // request
				nil, // response
				nil) // error
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

	// For generic content types, detect the actual image format from magic numbers
	finalContentType := contentType
	if strings.HasPrefix(contentType, "application/octet-stream") ||
		strings.HasPrefix(contentType, "binary/octet-stream") ||
		strings.HasPrefix(contentType, "application/binary") {
		if detectedType, isImage := p.detectImageFormat(imageData); isImage {
			finalContentType = detectedType
			logger.LogWithStructure(ctx, logger.LevelDebug, "Detected image format from magic numbers", map[string]interface{}{
				"original_content_type": contentType,
				"detected_content_type": detectedType,
				"url":                   imageURL,
			}, nil, nil, nil)
		} else {
			return "", fmt.Errorf("content type %s detected but data is not a valid image format", contentType)
		}
	}

	// Convert to base64 with data URL scheme
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", finalContentType, base64Data)

	logger.LogWithStructure(ctx, logger.LevelDebug, "Image downloaded and converted", map[string]interface{}{
		"original_url":          imageURL,
		"original_content_type": contentType,
		"final_content_type":    finalContentType,
		"size_bytes":            len(imageData),
		"base64_length":         len(base64Data),
		"data_url":              dataURL, // This will be properly truncated by LogWithStructure
	}, nil, nil, nil)

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
		"image/bmp",
		"image/tiff",
		"image/svg+xml",
	}

	// Check for explicit image content types
	for _, validType := range validTypes {
		if strings.HasPrefix(contentType, validType) {
			return true
		}
	}

	// Accept generic content types that might contain images
	// Many servers (like Telegram, Discord, etc.) return generic types for images
	genericTypes := []string{
		"application/octet-stream",
		"binary/octet-stream",
		"application/binary",
	}

	for _, genericType := range genericTypes {
		if strings.HasPrefix(contentType, genericType) {
			return true
		}
	}

	return false
}

// detectImageFormat detects image format from the first few bytes (magic numbers)
func (p *ImageProcessor) detectImageFormat(data []byte) (string, bool) {
	if len(data) < 12 {
		return "", false
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		return "image/png", true
	}

	// JPEG: FF D8 FF
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg", true
	}

	// GIF: 47 49 46 38 (GIF8)
	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif", true
	}

	// WebP: 52 49 46 46 ... 57 45 42 50 (RIFF...WEBP)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp", true
	}

	// BMP: 42 4D
	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return "image/bmp", true
	}

	// TIFF: 49 49 2A 00 (little endian) or 4D 4D 00 2A (big endian)
	if len(data) >= 4 {
		if (data[0] == 0x49 && data[1] == 0x49 && data[2] == 0x2A && data[3] == 0x00) ||
			(data[0] == 0x4D && data[1] == 0x4D && data[2] == 0x00 && data[3] == 0x2A) {
			return "image/tiff", true
		}
	}

	return "", false
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

// downloadAndConvertFileWithHeaders downloads a file from a URL with custom headers and converts it to text using markitdown
func (p *ImageProcessor) downloadAndConvertFileWithHeaders(ctx context.Context, fileURL string, headers map[string]string) (string, error) {
	logger.LogWithStructure(ctx, logger.LevelDebug, "Downloading file from URL with headers",
		map[string]interface{}{
			"url":     fileURL,
			"headers": headers,
		},
		nil, // request
		nil, // response
		nil) // error

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocks
	req.Header.Set("User-Agent", "Generative-API-Router/1.0")

	// Add custom headers if provided
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
			logger.LogWithStructure(ctx, logger.LevelDebug, "Added custom header for file download",
				map[string]interface{}{
					"header_key":   key,
					"header_value": value,
					"url":          fileURL,
				},
				nil, // request
				nil, // response
				nil) // error
		}
	}

	// Download the file
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: status %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("/tmp", "file_processor_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Read with size limit and write to temp file
	limitedReader := io.LimitReader(resp.Body, p.maxSize)
	fileData, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(fileData)) >= p.maxSize {
		return "", fmt.Errorf("file size exceeds limit of %d bytes", p.maxSize)
	}

	// Write data to temp file
	_, err = tempFile.Write(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tempFile.Close()

	// Convert file to text using markitdown
	textContent, err := p.convertFileToText(ctx, tempFile.Name(), fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to convert file to text: %w", err)
	}

	logger.LogWithStructure(ctx, logger.LevelDebug, "File downloaded and converted", map[string]interface{}{
		"original_url": fileURL,
		"content_type": resp.Header.Get("Content-Type"),
		"size_bytes":   len(fileData),
		"text_length":  len(textContent),
		"temp_file":    tempFile.Name(),
	}, nil, nil, nil)

	return textContent, nil
}

// convertFileToText converts a file to text using markitdown
func (p *ImageProcessor) convertFileToText(ctx context.Context, filePath, originalURL string) (string, error) {
	// Run markitdown command
	cmd := exec.CommandContext(ctx, "markitdown", filePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("markitdown failed: %w, stderr: %s", err, stderr.String())
	}

	// Get the text content
	textContent := stdout.String()

	// Create system message with file information
	fileInfo := map[string]interface{}{
		"source_url":   originalURL,
		"file_path":    filepath.Base(filePath),
		"content_size": len(textContent),
		"processed_by": "markitdown",
	}

	// Generate system message similar to generateImageFailureSystemMessage format
	systemMessage := p.generateFileSystemMessage(fileInfo, textContent)

	return systemMessage, nil
}

// generateFileSystemMessage creates a system message for successfully processed files
func (p *ImageProcessor) generateFileSystemMessage(fileInfo map[string]interface{}, content string) string {
	// Create file information summary
	sourceURL := fileInfo["source_url"].(string)
	contentSize := fileInfo["content_size"].(int)

	// Construct the system message
	systemMessage := fmt.Sprintf(`<system>
File successfully processed and converted to text from: %s (%d characters)

File content:
%s

The user cannot see this system message, so respond naturally based on the file content above as part of the ongoing conversation.
</system>`, sourceURL, contentSize, content)

	return systemMessage
}

// generateFileFailureSystemMessage creates a contextual system message for failed file downloads
func (p *ImageProcessor) generateFileFailureSystemMessage(err error, filePosition, totalFiles int, hasMixedScenario bool) string {
	// Determine the type of error for more specific messaging
	errorMsg := err.Error()
	var baseMessage string
	var contextPrefix string

	// Create context prefix for mixed scenarios
	if hasMixedScenario && totalFiles > 1 {
		contextPrefix = fmt.Sprintf("File %d of %d could not be processed. ", filePosition, totalFiles)
	} else if totalFiles > 1 {
		contextPrefix = fmt.Sprintf("One of the %d files provided could not be processed. ", totalFiles)
	} else {
		contextPrefix = "The file provided could not be processed. "
	}

	// Determine specific error message based on error type
	if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
		baseMessage = "Respond naturally that you couldn't access the file due to network connectivity issues. The file server appears to be unreachable or the domain doesn't exist. Ask the user to verify the URL or provide an alternative file."
	} else if strings.Contains(errorMsg, "status 401") || strings.Contains(errorMsg, "status 403") {
		baseMessage = "Respond naturally that the file requires authentication or access permissions that weren't provided. The file couldn't be accessed due to authorization issues. Suggest they provide proper authentication headers or use a publicly accessible file."
	} else if strings.Contains(errorMsg, "status 404") {
		baseMessage = "Respond naturally that the file URL appears to be broken or the file has been moved/deleted (404 Not Found). Ask them to provide a valid file URL."
	} else if strings.Contains(errorMsg, "size exceeds limit") {
		baseMessage = "Respond naturally that the file is too large to process (exceeds 20MB limit). Ask them to provide a smaller file or compress it before sharing."
	} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
		baseMessage = "Respond naturally that the file took too long to download due to slow response from the file server. Suggest they try again later or provide an alternative file."
	} else if strings.Contains(errorMsg, "markitdown failed") {
		baseMessage = "Respond naturally that the file couldn't be converted to text. The file format may not be supported by the text conversion tool, or the file may be corrupted. Ask them to provide the file in a different format (PDF, Word document, text file, etc.)."
	} else {
		// Generic error message for unknown error types
		baseMessage = "Respond naturally that there was a technical issue processing this file. Ask them to try providing the file again or use an alternative file."
	}

	// Add guidance for mixed scenarios
	var mixedScenarioGuidance string
	if hasMixedScenario && totalFiles > 1 {
		mixedScenarioGuidance = " You can still analyze and respond to the other files and images that were successfully processed."
	}

	// Construct the complete system message
	systemMessage := fmt.Sprintf("<system>\n%s%s%s The user cannot see this system message, so respond naturally as part of the ongoing conversation.\n</system>",
		contextPrefix, baseMessage, mixedScenarioGuidance)

	return systemMessage
}

package proxy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
	"github.com/aashari/go-generative-api-router/internal/utils"
)

// ImageProcessor handles image URL processing and conversion
type ImageProcessor struct {
	httpClient     *http.Client
	maxSize        int64
	fileProcessor  *FileProcessor
	audioProcessor *AudioProcessor
}

// NewImageProcessor creates a new image processor with default settings
func NewImageProcessor() *ImageProcessor {
	processor := &ImageProcessor{
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for image downloads
		},
		maxSize: 20 * 1024 * 1024, // 20MB limit
	}
	// Initialize file processor with all required fields
	processor.fileProcessor = &FileProcessor{
		imageProcessor: processor,
		audioProcessor: nil, // Will be set after audio processor is created
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for file downloads
		},
		maxSize: 20 * 1024 * 1024, // 20MB limit
	}
	// Initialize audio processor
	processor.audioProcessor = NewAudioProcessor()
	// Now set the audio processor in file processor
	processor.fileProcessor.audioProcessor = processor.audioProcessor
	return processor
}

// ContentPart represents a part of the message content
type ContentPart struct {
	Type       string      `json:"type"`
	Text       string      `json:"text,omitempty"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	FileURL    *FileURL    `json:"file_url,omitempty"`
	AudioURL   *AudioURL   `json:"audio_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
}

// ImageURL represents an image URL structure
type ImageURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// FileURL represents a file URL structure
type FileURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// AudioURL represents an audio URL structure for downloading
type AudioURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// InputAudio represents an audio input structure
type InputAudio struct {
	Data   string `json:"data"`   // Base64 encoded audio data
	Format string `json:"format"` // Format: "wav" or "mp3"
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

			// Extract file_url
			if fileURLVal, ok := itemMap["file_url"].(map[string]interface{}); ok {
				fileURL := &FileURL{}

				// Extract URL
				if urlStr, ok := fileURLVal["url"].(string); ok {
					fileURL.URL = urlStr
				}

				// Extract headers if present
				if headersVal, ok := fileURLVal["headers"].(map[string]interface{}); ok {
					headers := make(map[string]string)
					for key, value := range headersVal {
						if strValue, ok := value.(string); ok {
							headers[key] = strValue
						}
					}
					fileURL.Headers = headers
				}

				part.FileURL = fileURL
			}

			// Extract audio_url
			if audioURLVal, ok := itemMap["audio_url"].(map[string]interface{}); ok {
				audioURL := &AudioURL{}

				// Extract URL
				if urlStr, ok := audioURLVal["url"].(string); ok {
					audioURL.URL = urlStr
				}

				// Extract headers if present
				if headersVal, ok := audioURLVal["headers"].(map[string]interface{}); ok {
					headers := make(map[string]string)
					for key, value := range headersVal {
						if strValue, ok := value.(string); ok {
							headers[key] = strValue
						}
					}
					audioURL.Headers = headers
				}

				part.AudioURL = audioURL
			}

			// Extract input_audio
			if inputAudioVal, ok := itemMap["input_audio"].(map[string]interface{}); ok {
				inputAudio := &InputAudio{}

				// Extract data
				if dataStr, ok := inputAudioVal["data"].(string); ok {
					inputAudio.Data = dataStr
				}

				// Extract format
				if formatStr, ok := inputAudioVal["format"].(string); ok {
					inputAudio.Format = formatStr
				}

				part.InputAudio = inputAudio
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

		if part.Type == "file_url" && part.FileURL != nil {
			// Create file_url object without headers (headers are removed for vendor compatibility)
			fileURLMap := map[string]interface{}{
				"url": part.FileURL.URL,
			}
			partMap["file_url"] = fileURLMap
		}

		if part.Type == "audio_url" && part.AudioURL != nil {
			// Create audio_url object without headers (headers are removed for vendor compatibility)
			audioURLMap := map[string]interface{}{
				"url": part.AudioURL.URL,
			}
			partMap["audio_url"] = audioURLMap
		}

		if part.Type == "input_audio" && part.InputAudio != nil {
			// Create input_audio object
			inputAudioMap := map[string]interface{}{
				"data":   part.InputAudio.Data,
				"format": part.InputAudio.Format,
			}
			partMap["input_audio"] = inputAudioMap
		}

		result[i] = partMap
	}

	return result, nil
}

// processContentParts processes content parts concurrently with graceful error handling
func (p *ImageProcessor) processContentParts(ctx context.Context, parts []ContentPart) ([]ContentPart, error) {
	// Find all image URLs, files, and audio URLs that need processing
	itemsToProcess := make(map[int]int) // maps result index to parts index
	resultIndex := 0
	for i, part := range parts {
		if part.Type == "image_url" && part.ImageURL != nil && p.isPublicURL(part.ImageURL.URL) {
			itemsToProcess[resultIndex] = i
			resultIndex++
		} else if part.Type == "file_url" && part.FileURL != nil {
			// Process all file_url types without pre-validation
			itemsToProcess[resultIndex] = i
			resultIndex++
		} else if part.Type == "audio_url" && part.AudioURL != nil && p.isPublicURL(part.AudioURL.URL) {
			// Process all audio_url types
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
		if part.Type == "image_url" && part.ImageURL != nil {
			totalItems++
		} else if part.Type == "file_url" && part.FileURL != nil {
			totalItems++
		} else if part.Type == "audio_url" && part.AudioURL != nil {
			totalItems++
		}
	}

	// Log processing start (if logger is available)
	if len(itemsToProcess) > 0 {
		ctx = logger.WithComponent(ctx, "image_processor")
		ctx = logger.WithStage(ctx, "content_processing")
		logger.Info(ctx, "Processing image URLs, files, and audio URLs concurrently",
			"item_count", len(itemsToProcess),
			"total_parts", len(parts),
			"total_items", totalItems,
			"items_to_process", itemsToProcess)
	}

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
			} else if part.Type == "file_url" {
				// Process file using intelligent file processor
				fileContent, fileErr := p.fileProcessor.ProcessFileURLIntelligent(ctx, part.FileURL)
				err = fileErr
				if err == nil {
					processedContent = fileContent
				} else {
					// Error will be handled below
					processedContent = ContentPart{}
				}
			} else if part.Type == "audio_url" {
				// Process audio using modular audio processor
				audioData, audioErr := p.audioProcessor.ProcessAudioURL(ctx, part.AudioURL.URL, part.AudioURL.Headers)
				err = audioErr
				if err == nil {
					processedContent = ContentPart{
						Type: "input_audio",
						InputAudio: &InputAudio{
							Data:   audioData.Data,
							Format: audioData.Format,
						},
					}
				} else {
					// Error will be handled below
					processedContent = ContentPart{}
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
				if (parts[i].Type == "image_url" && parts[i].ImageURL != nil) || (parts[i].Type == "file_url" && parts[i].FileURL != nil) || (parts[i].Type == "audio_url" && parts[i].AudioURL != nil) {
					if i == result.Index {
						break
					}
					itemPosition++
				}
			}

			// Generate contextual failure message for failed item
			var failureMessage string
			if itemType == "file_url" {
				failureMessage = p.generateFileFailureMessage(result.Error, itemPosition, totalItems, len(itemsToProcess) > 1)
			} else if itemType == "audio_url" {
				failureMessage = p.generateAudioFailureMessage(result.Error, itemPosition, totalItems, len(itemsToProcess) > 1)
			} else {
				failureMessage = p.generateImageFailureMessage(result.Error, itemPosition, totalItems, len(itemsToProcess) > 1)
			}
			processedParts[result.Index] = ContentPart{
				Type: "text",
				Text: failureMessage,
			}

			ctx = logger.WithComponent(ctx, "image_processor")
			ctx = logger.WithStage(ctx, "error_handling")
			logger.Warn(ctx, "Item processing failed, replaced with failure message",
				"item_type", itemType,
				"item_index", result.Index,
				"item_position", itemPosition,
				"total_items", totalItems,
				"mixed_scenario", len(itemsToProcess) > 1,
				"error", result.Error.Error(),
				"failure_message", failureMessage)
		} else {
			processedParts[result.Index] = result.Content
		}
	}

	// Log processing completion with graceful handling summary
	ctx = logger.WithComponent(ctx, "image_processor")
	ctx = logger.WithStage(ctx, "completion_summary")
	logger.Info(ctx, "Item processing completed with graceful error handling",
		"processed_count", len(itemsToProcess),
		"successful_count", len(itemsToProcess)-len(errors),
		"failed_count", len(errors),
		"failed_item_indices", failedItems,
		"total_items", totalItems,
		"mixed_scenario", len(itemsToProcess) > 1 && len(errors) > 0 && len(errors) < len(itemsToProcess),
		"errors", errors,
		"graceful_handling", len(errors) > 0)

	// Always return success - errors are now handled gracefully
	return processedParts, nil
}

// extractFileURL safely extracts URL from FileURL struct, handling nil cases
func (p *ImageProcessor) extractFileURL(fileURL *FileURL) string {
	if fileURL == nil {
		return ""
	}
	return fileURL.URL
}

// extractFileHeaders safely extracts headers from FileURL struct, handling nil cases
func (p *ImageProcessor) extractFileHeaders(fileURL *FileURL) map[string]string {
	if fileURL == nil {
		return nil
	}
	return fileURL.Headers
}

// generateImageFailureMessage creates a contextual user message for failed image downloads
// generateProcessingFailureMessage creates a generic failure message for any item type (image, file, audio)
func (p *ImageProcessor) generateProcessingFailureMessage(err error, itemType string, itemPosition, totalItems int, hasMixedScenario bool) string {
	// Determine the type of error for more specific messaging
	errorMsg := err.Error()
	var baseMessage string
	var contextPrefix string

	// Create context prefix for mixed scenarios
	if hasMixedScenario && totalItems > 1 {
		contextPrefix = fmt.Sprintf("%s %d of %d could not be processed. ", strings.Title(itemType), itemPosition, totalItems)
	} else if totalItems > 1 {
		contextPrefix = fmt.Sprintf("One of the %d %ss provided could not be processed. ", totalItems, itemType)
	} else {
		contextPrefix = fmt.Sprintf("The %s provided could not be processed. ", itemType)
	}

	// Determine specific error message based on error type
	if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
		baseMessage = fmt.Sprintf("Respond naturally that you couldn't access the %s due to network connectivity issues. The %s server appears to be unreachable or the domain doesn't exist. Ask the user to verify the URL or provide an alternative %s.", itemType, itemType, itemType)
	} else if strings.Contains(errorMsg, "status 401") || strings.Contains(errorMsg, "status 403") {
		baseMessage = fmt.Sprintf("Respond naturally that the %s requires authentication or access permissions that weren't provided. The %s couldn't be accessed due to authorization issues. Suggest they provide proper authentication headers or use a publicly accessible %s.", itemType, itemType, itemType)
	} else if strings.Contains(errorMsg, "status 404") {
		baseMessage = fmt.Sprintf("Respond naturally that the %s URL appears to be broken or the %s has been moved/deleted (404 Not Found). Ask them to provide a valid %s URL.", itemType, itemType, itemType)
	} else if strings.Contains(errorMsg, "invalid content type") {
		var formatExamples string
		switch itemType {
		case "image":
			formatExamples = "(PNG, JPEG, GIF, WebP, etc.)"
		case "audio":
			formatExamples = "(MP3, WAV, etc.)"
		default:
			formatExamples = ""
		}
		baseMessage = fmt.Sprintf("Respond naturally that the URL doesn't point to a valid %s file. The content isn't an %s format that can be processed. Ask them to provide a direct link to an %s file %s.", itemType, itemType, itemType, formatExamples)
	} else if strings.Contains(errorMsg, "size exceeds limit") {
		baseMessage = fmt.Sprintf("Respond naturally that the %s file is too large to process (exceeds 20MB limit). Ask them to provide a smaller %s or compress it before sharing.", itemType, itemType)
	} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
		baseMessage = fmt.Sprintf("Respond naturally that the %s took too long to download due to slow response from the %s server. Suggest they try again later or provide an alternative %s.", itemType, itemType, itemType)
	} else if strings.Contains(errorMsg, "markitdown failed") && itemType == "file" {
		baseMessage = "Respond naturally that the file couldn't be converted to text. The file format may not be supported by the text conversion tool, or the file may be corrupted. Ask them to provide the file in a different format (PDF, Word document, text file, etc.)."
	} else {
		// Generic error message for unknown error types
		baseMessage = fmt.Sprintf("Respond naturally that there was a technical issue processing this %s. Ask them to try providing the %s again or use an alternative %s.", itemType, itemType, itemType)
	}

	// Add guidance for mixed scenarios
	var mixedScenarioGuidance string
	if hasMixedScenario && totalItems > 1 {
		if itemType == "file" {
			mixedScenarioGuidance = " You can still analyze and respond to the other files and images that were successfully processed."
		} else {
			mixedScenarioGuidance = fmt.Sprintf(" You can still analyze and respond to the other %ss that were successfully processed.", itemType)
		}
	}

	// Construct the complete user message (no system wrapper for vendor compatibility)
	userMessage := fmt.Sprintf("%s%s%s",
		contextPrefix, baseMessage, mixedScenarioGuidance)

	return userMessage
}

func (p *ImageProcessor) generateImageFailureMessage(err error, imagePosition, totalImages int, hasMixedScenario bool) string {
	return p.generateProcessingFailureMessage(err, "image", imagePosition, totalImages, hasMixedScenario)
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
	ctx = logger.WithComponent(ctx, "image_processor")
	ctx = logger.WithStage(ctx, "image_download")

	// Use the utility function to download the file
	imageData, contentType, err := utils.DownloadFile(ctx, imageURL, headers, p.maxSize)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}

	// Check content type
	if !p.isValidImageType(contentType) {
		return "", fmt.Errorf("invalid content type: %s", contentType)
	}

	// For generic content types, detect the actual image format from magic numbers
	finalContentType := contentType
	if strings.HasPrefix(contentType, "application/octet-stream") ||
		strings.HasPrefix(contentType, "binary/octet-stream") ||
		strings.HasPrefix(contentType, "application/binary") {
		if detectedType, isImage := p.detectImageFormat(imageData); isImage {
			finalContentType = detectedType
			logger.Debug(ctx, "Detected image format from magic numbers", "original_content_type", contentType, "detected_content_type", detectedType, "url", imageURL)
		} else {
			return "", fmt.Errorf("content type %s detected but data is not a valid image format", contentType)
		}
	}

	// Convert to base64 with data URL scheme
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", finalContentType, base64Data)

	logger.Debug(ctx, "Image downloaded and converted",
		"original_url", imageURL,
		"original_content_type", contentType,
		"final_content_type", finalContentType,
		"size_bytes", len(imageData),
		"base64_length", len(base64Data),
		"data_url", dataURL)

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

// detectDocumentFormat detects document format from the first few bytes (magic numbers)
func (p *ImageProcessor) detectDocumentFormat(data []byte) string {
	if len(data) < 16 {
		return "unknown"
	}

	// PDF: %PDF
	if len(data) >= 4 && data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
		return "application/pdf"
	}

	// ZIP-based formats (DOCX, XLSX, PPTX): PK (ZIP signature)
	if len(data) >= 2 && data[0] == 0x50 && data[1] == 0x4B {
		// Check for specific Office formats by looking deeper into the ZIP structure
		if len(data) >= 30 {
			content := string(data)
			if strings.Contains(content, "word/") || strings.Contains(content, "document.xml") {
				return "application/vnd.openxmlformats-officedocument.wordprocessingml.document" // DOCX
			} else if strings.Contains(content, "xl/") || strings.Contains(content, "workbook.xml") {
				return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" // XLSX
			} else if strings.Contains(content, "ppt/") || strings.Contains(content, "presentation.xml") {
				return "application/vnd.openxmlformats-officedocument.presentationml.presentation" // PPTX
			}
		}
		return "application/zip"
	}

	// Legacy Office formats
	// DOC, XLS, PPT: D0 CF 11 E0 A1 B1 1A E1 (OLE2 Compound Document)
	if len(data) >= 8 && data[0] == 0xD0 && data[1] == 0xCF && data[2] == 0x11 && data[3] == 0xE0 &&
		data[4] == 0xA1 && data[5] == 0xB1 && data[6] == 0x1A && data[7] == 0xE1 {
		// Could be DOC, XLS, or PPT - markitdown will handle the specifics
		return "application/msword" // Generic OLE2 document
	}

	// RTF: {\rtf
	if len(data) >= 5 && data[0] == 0x7B && data[1] == 0x5C && data[2] == 0x72 && data[3] == 0x74 && data[4] == 0x66 {
		return "application/rtf"
	}

	// Plain text files (check for common text patterns)
	if p.isLikelyTextFile(data) {
		return "text/plain"
	}

	// XML files: <?xml
	if len(data) >= 5 && data[0] == 0x3C && data[1] == 0x3F && data[2] == 0x78 && data[3] == 0x6D && data[4] == 0x6C {
		return "text/xml"
	}

	// HTML files: <!DOCTYPE html or <html
	content := strings.ToLower(string(data[:min(len(data), 100)]))
	if strings.Contains(content, "<!doctype html") || strings.Contains(content, "<html") {
		return "text/html"
	}

	// JSON files: starts with { or [
	trimmed := strings.TrimSpace(string(data[:min(len(data), 50)]))
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return utils.ContentTypeJSON
	}

	return "unknown"
}

// isLikelyTextFile checks if the data appears to be a text file
func (p *ImageProcessor) isLikelyTextFile(data []byte) bool {
	// Check first 512 bytes for text characteristics
	checkLength := min(len(data), 512)
	if checkLength == 0 {
		return false
	}

	// Count printable characters
	printableCount := 0
	for i := 0; i < checkLength; i++ {
		b := data[i]
		// Printable ASCII + common whitespace
		if (b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13 {
			printableCount++
		}
	}

	// If more than 95% of characters are printable, it's likely text
	return float64(printableCount)/float64(checkLength) > 0.95
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	ctx = logger.WithComponent(ctx, "image_processor")
	ctx = logger.WithStage(ctx, "file_download")

	// Use the utility function to download the file
	fileData, originalContentType, err := utils.DownloadFile(ctx, fileURL, headers, p.maxSize)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("/tmp", "file_processor_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write data to temp file
	_, err = tempFile.Write(fileData)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tempFile.Close()

	// Detect actual file type for better logging
	detectedFileType := p.detectDocumentFormat(fileData)

	// Convert file to text using markitdown
	textContent, err := p.convertFileToText(ctx, tempFile.Name(), fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to convert file to text: %w", err)
	}

	logger.Debug(ctx, "File downloaded and converted",
		"original_url", fileURL,
		"original_content_type", originalContentType,
		"detected_file_type", detectedFileType,
		"size_bytes", len(fileData),
		"text_length", len(textContent),
		"temp_file", tempFile.Name(),
		"content_type_detected", originalContentType != detectedFileType && detectedFileType != "unknown")

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

	// Generate user message for successful file processing
	userMessage := p.generateFileUserMessage(fileInfo, textContent)

	return userMessage, nil
}

// generateFileUserMessage creates a user message for successfully processed files
func (p *ImageProcessor) generateFileUserMessage(fileInfo map[string]interface{}, content string) string {
	// Create file information summary
	sourceURL := fileInfo["source_url"].(string)
	contentSize := fileInfo["content_size"].(int)

	// Construct user message with file content (no system wrapper for vendor compatibility)
	userMessage := fmt.Sprintf("File content from %s (%d characters):\n\n%s", sourceURL, contentSize, content)

	return userMessage
}

// generateFileFailureMessage creates a contextual user message for failed file downloads
func (p *ImageProcessor) generateFileFailureMessage(err error, filePosition, totalFiles int, hasMixedScenario bool) string {
	return p.generateProcessingFailureMessage(err, "file", filePosition, totalFiles, hasMixedScenario)
}

// generateAudioFailureMessage creates a contextual user message for failed audio downloads
func (p *ImageProcessor) generateAudioFailureMessage(err error, audioPosition, totalAudios int, hasMixedScenario bool) string {
	return p.generateProcessingFailureMessage(err, "audio", audioPosition, totalAudios, hasMixedScenario)
}

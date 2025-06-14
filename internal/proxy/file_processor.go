package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aashari/go-generative-api-router/internal/logger"
)

// FileProcessor handles file processing operations with intelligent routing
type FileProcessor struct {
	imageProcessor *ImageProcessor
	audioProcessor *AudioProcessor
	httpClient     *http.Client
	maxSize        int64
}

// NewFileProcessor creates a new file processor
func NewFileProcessor() *FileProcessor {
	return &FileProcessor{
		imageProcessor: NewImageProcessor(),
		audioProcessor: NewAudioProcessor(),
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for file downloads
		},
		maxSize: 20 * 1024 * 1024, // 20MB limit
	}
}

// ProcessFileURL processes a file URL and returns the processed content
func (f *FileProcessor) ProcessFileURL(ctx context.Context, fileURL *FileURL) (string, error) {
	if fileURL == nil {
		return "", fmt.Errorf("file_url is nil")
	}

	url := fileURL.URL
	headers := fileURL.Headers

	// Empty URL should return a clear error message without system wrapper
	if url == "" {
		return "Error: No file URL provided. Please provide a valid file URL to process.", nil
	}

	// First, detect the file type by downloading headers and initial content
	fileType, err := f.detectFileType(ctx, url, headers)
	if err != nil {
		ctx = logger.WithComponent(ctx, "file_processor")
		ctx = logger.WithStage(ctx, "file_type_detection")
		logger.Debug(ctx, "Failed to detect file type", "url", url, "error", err.Error())
		// Fall back to default file processing
		return f.processAsDocument(ctx, url, headers)
	}

	ctx = logger.WithComponent(ctx, "file_processor")
	ctx = logger.WithStage(ctx, "file_type_detection")
	logger.Info(ctx, "Detected file type for intelligent routing", "url", url, "file_type", fileType)

	// Route based on detected file type
	switch fileType {
	case "image":
		return f.processAsImage(ctx, url, headers)
	case "audio":
		return f.processAsAudio(ctx, url, headers)
	default:
		return f.processAsDocument(ctx, url, headers)
	}
}

// ProcessFileURLIntelligent processes a file URL and returns the appropriate ContentPart type
func (f *FileProcessor) ProcessFileURLIntelligent(ctx context.Context, fileURL *FileURL) (ContentPart, error) {
	if fileURL == nil {
		return ContentPart{}, fmt.Errorf("file_url is nil")
	}

	url := fileURL.URL
	headers := fileURL.Headers

	// Empty URL should return a clear error message
	if url == "" {
		return ContentPart{
			Type: "text",
			Text: "Error: No file URL provided. Please provide a valid file URL to process.",
		}, nil
	}

	// First, detect the file type by downloading headers and initial content
	fileType, err := f.detectFileType(ctx, url, headers)
	if err != nil {
		ctx = logger.WithComponent(ctx, "file_processor")
		ctx = logger.WithStage(ctx, "intelligent_routing")
		logger.Debug(ctx, "Failed to detect file type for intelligent routing", "url", url, "error", err.Error())
		// Fall back to default file processing as text
		content, procErr := f.processAsDocument(ctx, url, headers)
		if procErr != nil {
			return ContentPart{}, procErr
		}
		return ContentPart{
			Type: "text",
			Text: content,
		}, nil
	}

	ctx = logger.WithComponent(ctx, "file_processor")
	ctx = logger.WithStage(ctx, "intelligent_routing")
	logger.Info(ctx, "Detected file type for intelligent routing", "url", url, "file_type", fileType)

	// Route based on detected file type
	switch fileType {
	case "image":
		// Process as image and return image_url ContentPart
		dataURL, err := f.imageProcessor.downloadAndConvertImageWithHeaders(ctx, url, headers)
		if err != nil {
			// Return error as text
			return ContentPart{
				Type: "text",
				Text: f.imageProcessor.generateImageFailureMessage(err, 1, 1, false),
			}, nil
		}
		// Return as image_url ContentPart
		return ContentPart{
			Type: "image_url",
			ImageURL: &ImageURL{
				URL: dataURL,
			},
		}, nil

	case "audio":
		// Process as audio and return input_audio ContentPart
		audioData, err := f.audioProcessor.ProcessAudioURL(ctx, url, headers)
		if err != nil {
			// Return error as text
			errorMsg := err.Error()
			var errorText string
			if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
				errorText = fmt.Sprintf("I couldn't access the audio at %s due to network connectivity issues.", url)
			} else if strings.Contains(errorMsg, "status 404") {
				errorText = fmt.Sprintf("The audio URL %s appears to be broken or the file has been moved/deleted.", url)
			} else {
				errorText = fmt.Sprintf("There was a technical issue processing the audio at %s.", url)
			}
			return ContentPart{
				Type: "text",
				Text: errorText,
			}, nil
		}
		// Return as input_audio ContentPart
		return ContentPart{
			Type: "input_audio",
			InputAudio: &InputAudio{
				Data:   audioData.Data,
				Format: audioData.Format,
			},
		}, nil

	default:
		// Process as document and return as text
		content, err := f.processAsDocument(ctx, url, headers)
		if err != nil {
			return ContentPart{}, err
		}
		return ContentPart{
			Type: "text",
			Text: content,
		}, nil
	}
}

// detectFileType downloads the beginning of the file to determine its type
func (f *FileProcessor) detectFileType(ctx context.Context, fileURL string, headers map[string]string) (string, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Generative-API-Router/1.0")

	// Add custom headers if provided
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// Make the request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch file: status %d", resp.StatusCode)
	}

	// Get content type from header
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	// Read first 512 bytes for magic number detection
	buffer := make([]byte, 512)
	n, err := io.ReadFull(resp.Body, buffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	buffer = buffer[:n]

	// Check content type first
	if f.isImageContentType(contentType) {
		return "image", nil
	}
	if f.isAudioContentType(contentType) {
		return "audio", nil
	}

	// Check magic numbers if content type is generic
	if f.detectImageFormat(buffer) != "" {
		return "image", nil
	}
	if f.detectAudioFormat(buffer) != "" {
		return "audio", nil
	}

	// Default to document
	return "document", nil
}

// isImageContentType checks if the content type indicates an image
func (f *FileProcessor) isImageContentType(contentType string) bool {
	imageTypes := []string{
		"image/png", "image/jpeg", "image/jpg", "image/gif",
		"image/webp", "image/bmp", "image/tiff", "image/svg+xml",
	}
	for _, imageType := range imageTypes {
		if strings.HasPrefix(contentType, imageType) {
			return true
		}
	}
	return false
}

// isAudioContentType checks if the content type indicates an audio file
func (f *FileProcessor) isAudioContentType(contentType string) bool {
	audioTypes := []string{
		"audio/mpeg", "audio/mp3", "audio/wav", "audio/wave",
		"audio/x-wav", "audio/ogg", "audio/flac", "audio/aac",
		"audio/mp4", "audio/webm", "audio/3gpp", "audio/3gpp2",
	}
	for _, audioType := range audioTypes {
		if strings.HasPrefix(contentType, audioType) {
			return true
		}
	}
	return false
}

// detectImageFormat detects image format from magic numbers
func (f *FileProcessor) detectImageFormat(data []byte) string {
	// Reuse the image processor's detection logic
	if len(data) < 12 {
		return ""
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// JPEG: FF D8 FF
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// GIF: 47 49 46 38 (GIF8)
	if len(data) >= 4 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}

	// WebP: 52 49 46 46 ... 57 45 42 50 (RIFF...WEBP)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}

	// BMP: 42 4D
	if len(data) >= 2 && data[0] == 0x42 && data[1] == 0x4D {
		return "image/bmp"
	}

	return ""
}

// detectAudioFormat detects audio format from magic numbers
func (f *FileProcessor) detectAudioFormat(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	// MP3: ID3 tag (49 44 33) or MPEG sync (FF FB, FF FA, FF F3, FF F2)
	if len(data) >= 3 && data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33 {
		return "audio/mp3"
	}
	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "audio/mp3"
	}

	// WAV: RIFF ... WAVE
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "audio/wav"
	}

	// OGG: OggS
	if len(data) >= 4 && data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		return "audio/ogg"
	}

	// FLAC: fLaC
	if len(data) >= 4 && data[0] == 0x66 && data[1] == 0x4C && data[2] == 0x61 && data[3] == 0x43 {
		return "audio/flac"
	}

	// M4A/AAC: ftyp
	if len(data) >= 8 && data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		return "audio/mp4"
	}

	return ""
}

// processAsImage processes the file as an image
func (f *FileProcessor) processAsImage(ctx context.Context, url string, headers map[string]string) (string, error) {
	ctx = logger.WithComponent(ctx, "file_processor")
	ctx = logger.WithStage(ctx, "image_processing")
	logger.Info(ctx, "Processing file_url as image", "url", url)

	// Use the image processor to download and convert to base64
	_, err := f.imageProcessor.downloadAndConvertImageWithHeaders(ctx, url, headers)
	if err != nil {
		// Return user-friendly error message
		return f.imageProcessor.generateImageFailureMessage(err, 1, 1, false), nil
	}

	// Return as a user message indicating it's an image
	return fmt.Sprintf("I've received an image from: %s", url), nil
}

// processAsAudio processes the file as audio
func (f *FileProcessor) processAsAudio(ctx context.Context, url string, headers map[string]string) (string, error) {
	ctx = logger.WithComponent(ctx, "file_processor")
	ctx = logger.WithStage(ctx, "audio_processing")
	logger.Info(ctx, "Processing file_url as audio", "url", url)

	// Use the audio processor to download and convert
	audioData, err := f.audioProcessor.ProcessAudioURL(ctx, url, headers)
	if err != nil {
		// Return user-friendly error message
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
			return fmt.Sprintf("I couldn't access the audio at %s due to network connectivity issues. The server appears to be unreachable.", url), nil
		} else if strings.Contains(errorMsg, "status 404") {
			return fmt.Sprintf("The audio URL %s appears to be broken or the file has been moved/deleted.", url), nil
		} else {
			return fmt.Sprintf("There was a technical issue processing the audio at %s. Please try providing the audio again.", url), nil
		}
	}

	// Return as a user message indicating audio was received
	return fmt.Sprintf("I've received an audio file (%s format) from: %s", audioData.Format, url), nil
}

// processAsDocument processes the file as a document using markitdown
func (f *FileProcessor) processAsDocument(ctx context.Context, url string, headers map[string]string) (string, error) {
	ctx = logger.WithComponent(ctx, "file_processor")
	ctx = logger.WithStage(ctx, "document_processing")
	logger.Info(ctx, "Processing file_url as document", "url", url)

	// Use the existing markitdown processing
	content, err := f.imageProcessor.downloadAndConvertFileWithHeaders(ctx, url, headers)
	if err != nil {
		// Return user-friendly error message instead of technical error
		return f.generateFileErrorMessage(err, url), nil
	}

	return content, nil
}

// generateFileErrorMessage creates a user-friendly error message for file processing failures
func (f *FileProcessor) generateFileErrorMessage(err error, fileURL string) string {
	errorMsg := err.Error()

	// Determine specific error message based on error type
	if strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "dial tcp") {
		return fmt.Sprintf("I couldn't access the file at %s due to network connectivity issues. The file server appears to be unreachable or the domain doesn't exist. Please verify the URL or provide an alternative file.", fileURL)
	} else if strings.Contains(errorMsg, "status 401") || strings.Contains(errorMsg, "status 403") {
		return fmt.Sprintf("The file at %s requires authentication or access permissions that weren't provided. Please provide proper authentication headers or use a publicly accessible file.", fileURL)
	} else if strings.Contains(errorMsg, "status 404") {
		return fmt.Sprintf("The file URL %s appears to be broken or the file has been moved/deleted (404 Not Found). Please provide a valid file URL.", fileURL)
	} else if strings.Contains(errorMsg, "size exceeds limit") {
		return fmt.Sprintf("The file at %s is too large to process (exceeds 20MB limit). Please provide a smaller file or compress it before sharing.", fileURL)
	} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
		return fmt.Sprintf("The file at %s took too long to download due to slow response from the file server. Please try again later or provide an alternative file.", fileURL)
	} else if strings.Contains(errorMsg, "markitdown failed") {
		return fmt.Sprintf("The file at %s couldn't be converted to text. The file format may not be supported by the text conversion tool, or the file may be corrupted. Please provide the file in a different format (PDF, Word document, text file, etc.).", fileURL)
	} else {
		// Generic error message for unknown error types
		return fmt.Sprintf("There was a technical issue processing the file at %s. Please try providing the file again or use an alternative file.", fileURL)
	}
}

// IsFileURLSupported checks if the file URL can be processed
func (f *FileProcessor) IsFileURLSupported(fileURL *FileURL) bool {
	if fileURL == nil || fileURL.URL == "" {
		return false
	}
	// Accept any URL - we'll intelligently route based on content
	return true
}

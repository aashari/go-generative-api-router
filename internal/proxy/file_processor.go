package proxy

import (
	"context"
	"fmt"
	"strings"
)

// FileProcessor handles file processing operations
type FileProcessor struct {
	imageProcessor *ImageProcessor
}

// NewFileProcessor creates a new file processor
func NewFileProcessor() *FileProcessor {
	return &FileProcessor{
		imageProcessor: NewImageProcessor(),
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

	// Process the file using markitdown
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
	// Accept any URL - let markitdown handle format validation
	return true
}

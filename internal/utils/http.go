package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DownloadFile downloads a file from a URL with optional headers and size limit
func DownloadFile(ctx context.Context, url string, headers map[string]string, maxSize int64) ([]byte, string, error) {

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocks
	req.Header.Set(HeaderUserAgent, ServiceName)

	// Add custom headers if provided
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	// Download the file
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download file: status %d", resp.StatusCode)
	}

	// Get content type
	contentType := resp.Header.Get(HeaderContentType)

	// Read with size limit
	limitedReader := io.LimitReader(resp.Body, maxSize)
	fileData, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Check if we hit the size limit
	if int64(len(fileData)) >= maxSize {
		return nil, "", fmt.Errorf("file size exceeds limit of %d bytes", maxSize)
	}

	// File downloaded successfully

	return fileData, contentType, nil
}

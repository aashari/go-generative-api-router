package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		expectedContent := "Hello, World!"
		expectedContentType := "text/plain"

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify User-Agent header
			assert.Equal(t, ServiceName, r.Header.Get("User-Agent"))

			w.Header().Set("Content-Type", expectedContentType)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedContent))
		}))
		defer server.Close()

		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(data))
		assert.Equal(t, expectedContentType, contentType)
	})

	t.Run("download with custom headers", func(t *testing.T) {
		expectedAuth := "Bearer test-token"
		expectedCustom := "custom-value"

		// Create a test server that checks headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify custom headers were set
			assert.Equal(t, expectedAuth, r.Header.Get("Authorization"))
			assert.Equal(t, expectedCustom, r.Header.Get("X-Custom-Header"))
			assert.Equal(t, ServiceName, r.Header.Get("User-Agent"))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()

		headers := map[string]string{
			"Authorization":   expectedAuth,
			"X-Custom-Header": expectedCustom,
		}

		ctx := context.Background()
		_, _, err := DownloadFile(ctx, server.URL, headers, 1024)

		require.NoError(t, err)
	})

	t.Run("404 not found", func(t *testing.T) {
		// Create a test server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
		assert.Nil(t, data)
		assert.Empty(t, contentType)
	})

	t.Run("500 internal server error", func(t *testing.T) {
		// Create a test server that returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.Nil(t, data)
		assert.Empty(t, contentType)
	})

	t.Run("file size exceeds limit", func(t *testing.T) {
		largeContent := make([]byte, 2048) // 2KB content
		for i := range largeContent {
			largeContent[i] = 'A'
		}

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(largeContent)
		}))
		defer server.Close()

		ctx := context.Background()
		maxSize := int64(1024) // 1KB limit
		data, contentType, err := DownloadFile(ctx, server.URL, nil, maxSize)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file size exceeds limit")
		assert.Nil(t, data)
		assert.Empty(t, contentType)
	})

	t.Run("context timeout", func(t *testing.T) {
		// Create a test server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()

		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
		assert.Nil(t, data)
		assert.Empty(t, contentType)
	})

	t.Run("invalid URL", func(t *testing.T) {
		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, "not-a-valid-url", nil, 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download file")
		assert.Nil(t, data)
		assert.Empty(t, contentType)
	})

	t.Run("server not reachable", func(t *testing.T) {
		ctx := context.Background()
		// Use a URL that should not be reachable
		data, contentType, err := DownloadFile(ctx, "http://localhost:99999/nonexistent", nil, 1024)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download file")
		assert.Nil(t, data)
		assert.Empty(t, contentType)
	})

	t.Run("empty response", func(t *testing.T) {
		// Create a test server that returns empty content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			// Don't write any content
		}))
		defer server.Close()

		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		require.NoError(t, err)
		assert.Empty(t, data)
		assert.Equal(t, "text/plain", contentType)
	})

	t.Run("binary content", func(t *testing.T) {
		// Test with binary data
		binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
		expectedContentType := "image/png"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", expectedContentType)
			w.WriteHeader(http.StatusOK)
			w.Write(binaryData)
		}))
		defer server.Close()

		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		require.NoError(t, err)
		assert.Equal(t, binaryData, data)
		assert.Equal(t, expectedContentType, contentType)
	})

	t.Run("exact size limit", func(t *testing.T) {
		// Test when content is exactly at the size limit
		content := make([]byte, 1024) // Exactly 1KB
		for i := range content {
			content[i] = 'B'
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(content)
		}))
		defer server.Close()

		ctx := context.Background()
		maxSize := int64(1024) // Exactly 1KB limit
		_, _, err := DownloadFile(ctx, server.URL, nil, maxSize)

		// At exactly the limit, it should still error due to >= check
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file size exceeds limit")
	})

	t.Run("missing content-type header", func(t *testing.T) {
		// Create a test server that doesn't set Content-Type
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}))
		defer server.Close()

		ctx := context.Background()
		data, contentType, err := DownloadFile(ctx, server.URL, nil, 1024)

		require.NoError(t, err)
		assert.Equal(t, "OK", string(data))
		// The test server sets Content-Type automatically
		assert.Contains(t, contentType, "text/plain")
	})
}

func TestDownloadFileEdgeCases(t *testing.T) {
	t.Run("very large maxSize", func(t *testing.T) {
		content := "small content"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}))
		defer server.Close()

		ctx := context.Background()
		// Set a very large max size
		maxSize := int64(1024 * 1024 * 1024) // 1GB
		data, _, err := DownloadFile(ctx, server.URL, nil, maxSize)

		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("zero maxSize", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("any content"))
		}))
		defer server.Close()

		ctx := context.Background()
		maxSize := int64(0)
		data, _, err := DownloadFile(ctx, server.URL, nil, maxSize)

		// Should immediately exceed the limit
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file size exceeds limit")
		assert.Nil(t, data)
	})

	t.Run("redirect handling", func(t *testing.T) {
		finalContent := "final content"

		// Create final server
		finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(finalContent))
		}))
		defer finalServer.Close()

		// Create redirect server
		redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, finalServer.URL, http.StatusFound)
		}))
		defer redirectServer.Close()

		ctx := context.Background()
		data, _, err := DownloadFile(ctx, redirectServer.URL, nil, 1024)

		// HTTP client should handle redirects automatically
		require.NoError(t, err)
		assert.Equal(t, finalContent, string(data))
	})
}

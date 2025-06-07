package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateBase64InData(t *testing.T) {
	// Create a long base64 string (longer than 100 characters)
	longBase64 := strings.Repeat("ABCDEFGHIJ", 20) // 200 characters
	dataURL := "data:image/png;base64," + longBase64

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "truncate long base64 string",
			input:    dataURL,
			expected: "data:image/png;base64," + longBase64[:50] + "...[100 chars truncated]..." + longBase64[len(longBase64)-50:],
		},
		{
			name:     "short base64 string unchanged",
			input:    "data:image/png;base64,shortstring",
			expected: "data:image/png;base64,shortstring",
		},
		{
			name:     "non-data URL unchanged",
			input:    "https://example.com/image.jpg",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "regular string unchanged",
			input:    "just a regular string",
			expected: "just a regular string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateBase64InData(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateBase64InDataStructures(t *testing.T) {
	// Create a long base64 string
	longBase64 := strings.Repeat("ABCDEFGHIJ", 20) // 200 characters
	dataURL := "data:image/png;base64," + longBase64
	expectedTruncated := "data:image/png;base64," + longBase64[:50] + "...[100 chars truncated]..." + longBase64[len(longBase64)-50:]

	tests := []struct {
		name     string
		input    interface{}
		validate func(t *testing.T, result interface{})
	}{
		{
			name: "map with base64 data",
			input: map[string]interface{}{
				"image_url": dataURL,
				"other":     "normal data",
			},
			validate: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				assert.Equal(t, expectedTruncated, resultMap["image_url"])
				assert.Equal(t, "normal data", resultMap["other"])
			},
		},
		{
			name: "slice with base64 data",
			input: []interface{}{
				dataURL,
				"normal string",
			},
			validate: func(t *testing.T, result interface{}) {
				resultSlice := result.([]interface{})
				assert.Equal(t, expectedTruncated, resultSlice[0])
				assert.Equal(t, "normal string", resultSlice[1])
			},
		},
		{
			name: "nested structure",
			input: map[string]interface{}{
				"messages": []interface{}{
					map[string]interface{}{
						"content": []interface{}{
							map[string]interface{}{
								"type": "image_url",
								"image_url": map[string]interface{}{
									"url": dataURL,
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				messages := resultMap["messages"].([]interface{})
				content := messages[0].(map[string]interface{})["content"].([]interface{})
				imageURL := content[0].(map[string]interface{})["image_url"].(map[string]interface{})
				assert.Equal(t, expectedTruncated, imageURL["url"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateBase64InData(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestTruncateBase64String(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exactly 100 chars - no truncation",
			input:    "data:image/png;base64," + strings.Repeat("A", 100),
			expected: "data:image/png;base64," + strings.Repeat("A", 100),
		},
		{
			name:     "101 chars - truncation applied",
			input:    "data:image/png;base64," + strings.Repeat("A", 101),
			expected: "data:image/png;base64," + strings.Repeat("A", 50) + "...[1 chars truncated]..." + strings.Repeat("A", 50),
		},
		{
			name:     "200 chars - truncation applied",
			input:    "data:image/png;base64," + strings.Repeat("B", 200),
			expected: "data:image/png;base64," + strings.Repeat("B", 50) + "...[100 chars truncated]..." + strings.Repeat("B", 50),
		},
		{
			name:     "not a data URL",
			input:    "https://example.com/image.jpg",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "data URL without base64",
			input:    "data:text/plain;charset=utf-8,Hello%20World",
			expected: "data:text/plain;charset=utf-8,Hello%20World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateBase64String(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

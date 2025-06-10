package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const apiBase = "http://localhost:8082"

// ContentPart represents a part of multi-part content
type ContentPart struct {
	Type    string   `json:"type"`
	Text    string   `json:"text,omitempty"`
	FileURL *FileURL `json:"file_url,omitempty"`
}

// FileURL represents a file URL with optional headers
type FileURL struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Message can contain either string content or multi-part content
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

func chatCompletion(message string) (string, error) {
	req := ChatRequest{
		Model: "any-model",
		Messages: []Message{
			{Role: "user", Content: message},
		},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(apiBase+"/v1/chat/completions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	return string(result), nil
}

func processFile(fileURL, question string, headers map[string]string) (string, error) {
	content := []ContentPart{
		{
			Type: "text",
			Text: question,
		},
		{
			Type: "file_url",
			FileURL: &FileURL{
				URL:     fileURL,
				Headers: headers,
			},
		},
	}

	req := ChatRequest{
		Model: "document-analyzer",
		Messages: []Message{
			{Role: "user", Content: content},
		},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(apiBase+"/v1/chat/completions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	return string(result), nil
}

func processMultipleFiles(fileURLs []string, question string) (string, error) {
	content := []ContentPart{
		{
			Type: "text",
			Text: question,
		},
	}

	// Add each file to the content
	for _, url := range fileURLs {
		content = append(content, ContentPart{
			Type: "file_url",
			FileURL: &FileURL{
				URL: url,
			},
		})
	}

	req := ChatRequest{
		Model: "multi-file-analyzer",
		Messages: []Message{
			{Role: "user", Content: content},
		},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(apiBase+"/v1/chat/completions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	return string(result), nil
}

func main() {
	fmt.Println("=== Basic Chat Example ===")
	result, err := chatCompletion("Hello, how are you?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println(result)
	}

	fmt.Println("\n=== File Processing Example ===")
	// Example with Apple's research paper
	fileResult, err := processFile(
		"https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf",
		"Please provide a brief summary of this research paper.",
		nil, // no custom headers
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println(fileResult)
	}

	fmt.Println("\n=== Multiple Files Example ===")
	// Example with multiple files
	multiResult, err := processMultipleFiles([]string{
		"https://ml-site.cdn-apple.com/papers/the-illusion-of-thinking.pdf",
		"https://example.com/another-document.pdf", // Would need a real URL
	}, "Compare these two documents.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println(multiResult)
	}
}

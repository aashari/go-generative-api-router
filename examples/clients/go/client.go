package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const apiBase = "http://localhost:8082"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

func main() {
	req := ChatRequest{
		Model: "any-model",
		Messages: []Message{
			{Role: "user", Content: "Hello, how are you?"},
		},
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(apiBase+"/v1/chat/completions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	fmt.Println(string(result))
}

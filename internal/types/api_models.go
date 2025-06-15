package types

// ChatCompletionRequest represents a request to the chat completions API
type ChatCompletionRequest struct {
	Messages []Message `json:"messages" example:"[]"`
	Model    string    `json:"model" example:"gpt-4o"`
	Stream   bool      `json:"stream,omitempty" example:"false"`
	// Added OpenAI-compatible fields
	MaxTokens        int                  `json:"max_tokens,omitempty" example:"100"`
	Temperature      float64              `json:"temperature,omitempty" example:"0.7"`
	TopP             float64              `json:"top_p,omitempty" example:"1"`
	N                int                  `json:"n,omitempty" example:"1"`
	Stop             []string             `json:"stop,omitempty"`
	PresencePenalty  float64              `json:"presence_penalty,omitempty" example:"0"`
	FrequencyPenalty float64              `json:"frequency_penalty,omitempty" example:"0"`
	LogitBias        map[string]float64   `json:"logit_bias,omitempty"`
	User             string               `json:"user,omitempty" example:"user-123"`
	Functions        []FunctionDefinition `json:"functions,omitempty"`
	FunctionCall     string               `json:"function_call,omitempty" example:"auto"`
	Tools            []Tool               `json:"tools,omitempty"`
	ToolChoice       string               `json:"tool_choice,omitempty" example:"auto"`
	ResponseFormat   map[string]string    `json:"response_format,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role" example:"user"`
	Content    string     `json:"content" example:"Hello, how are you?"`
	Name       string     `json:"name,omitempty" example:"John"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// FunctionDefinition represents an available function definition
type FunctionDefinition struct {
	Name        string                 `json:"name" example:"get_weather"`
	Description string                 `json:"description,omitempty" example:"Get the current weather in a location"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Tool represents a tool available to the model
type Tool struct {
	Type     string                 `json:"type" example:"function"`
	Function map[string]interface{} `json:"function,omitempty"`
}

// ToolCall represents a call to a tool
type ToolCall struct {
	ID       string                 `json:"id" example:"call_8qty38"`
	Type     string                 `json:"type" example:"function"`
	Function map[string]interface{} `json:"function"`
}

// ChatCompletionResponse represents a response from the chat completions API
type ChatCompletionResponse struct {
	ID                string   `json:"id" example:"chatcmpl-abc123"`
	Object            string   `json:"object" example:"chat.completion"`
	Created           int64    `json:"created" example:"1677652288"`
	Model             string   `json:"model" example:"gpt-4o"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty" example:"fp_abc123"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	ServiceTier       string   `json:"service_tier,omitempty" example:"default"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index" example:"0"`
	Message      Message `json:"message"`
	LogProbs     string  `json:"logprobs" example:"null"`
	FinishReason string  `json:"finish_reason" example:"stop"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens" example:"10"`
	CompletionTokens int `json:"completion_tokens" example:"20"`
	TotalTokens      int `json:"total_tokens" example:"30"`
	// Added additional usage details for OpenAI compatibility
	PromptTokensDetails     TokenDetails `json:"prompt_tokens_details"`
	CompletionTokensDetails TokenDetails `json:"completion_tokens_details"`
}

// TokenDetails represents detailed token usage information
type TokenDetails struct {
	CachedTokens             int `json:"cached_tokens" example:"0"`
	AudioTokens              int `json:"audio_tokens" example:"0"`
	ReasoningTokens          int `json:"reasoning_tokens,omitempty" example:"0"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty" example:"0"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty" example:"0"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// ErrorInfo contains details about an error
type ErrorInfo struct {
	Message string `json:"message" example:"Invalid model specified"`
	Type    string `json:"type" example:"invalid_request_error"`
	Param   string `json:"param" example:"model"`
	Code    string `json:"code,omitempty" example:"invalid_model"`
}

// ModelsResponse represents the response from the models endpoint
type ModelsResponse struct {
	Object string  `json:"object" example:"list"`
	Data   []Model `json:"data"`
}

// Model represents a language model
type Model struct {
	ID      string `json:"id" example:"gpt-4o"`
	Object  string `json:"object" example:"model"`
	Created int64  `json:"created" example:"1677610602"`
	OwnedBy string `json:"owned_by" example:"openai"`
}

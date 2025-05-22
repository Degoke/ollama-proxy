package main

import (
	// "bytes"
	"net/http"
)

// RequestDetails contains information about the incoming request
type RequestDetails struct {
	APIKey           string            `json:"apiKey"`
	IPAddress        string            `json:"ipAddress"`
	UserAgent        string            `json:"userAgent"`
	Headers          map[string]string `json:"headers"`
	Model            string            `json:"model"`
	InputTokenLength int               `json:"inputTokenLength"`
	Endpoint         string            `json:"endpoint"`
}

// ValidationResponse represents the response from the external validation server
type ValidationResponse struct {
	Valid       bool `json:"valid"`
	RateLimited bool `json:"rateLimited"`
}

// MetricsData contains information to be sent to the metrics server
type MetricsData struct {
	APIKey            string `json:"apiKey"`
	Model             string `json:"model"`
	InputTokenLength  int    `json:"inputTokenLength"`
	OutputTokenLength int    `json:"outputTokenLength"`
	RequestDurationMs int64  `json:"requestDurationMs"`
	Endpoint          string `json:"endpoint"`
}

// ChatRequest represents the structure of a chat request to Ollama
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   interface{}   `json:"format,omitempty"`
	Options  interface{}   `json:"options,omitempty"`
}

// ChatMessage represents a single message in a chat request
type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call in a chat message
type ToolCall struct {
	Function struct {
		Name      string      `json:"name"`
		Arguments interface{} `json:"arguments"`
	} `json:"function"`
}

// GenerateRequest represents the structure of a generate request to Ollama
type GenerateRequest struct {
	Model   string      `json:"model"`
	Prompt  string      `json:"prompt"`
	Stream  bool        `json:"stream"`
	Format  interface{} `json:"format,omitempty"`
	Options interface{} `json:"options,omitempty"`
	Images  []string    `json:"images,omitempty"`
}

// EmbedRequest represents the structure of an embedding request to Ollama
type EmbedRequest struct {
	Model   string      `json:"model"`
	Input   interface{} `json:"input"`
	Options interface{} `json:"options,omitempty"`
}

// CreateRequest represents the structure of a model creation request
type CreateRequest struct {
	Model      string            `json:"model"`
	From       string            `json:"from,omitempty"`
	Files      map[string]string `json:"files,omitempty"`
	Adapters   map[string]string `json:"adapters,omitempty"`
	Template   string            `json:"template,omitempty"`
	License    interface{}       `json:"license,omitempty"`
	System     string            `json:"system,omitempty"`
	Parameters interface{}       `json:"parameters,omitempty"`
	Messages   []ChatMessage     `json:"messages,omitempty"`
	Stream     bool              `json:"stream,omitempty"`
	Quantize   string            `json:"quantize,omitempty"`
}

// ChatResponse represents the structure of a chat response from Ollama
type ChatResponse struct {
	Model           string      `json:"model"`
	CreatedAt       string      `json:"created_at"`
	Message         ChatMessage `json:"message"`
	Done            bool        `json:"done"`
	DoneReason      string      `json:"done_reason,omitempty"`
	TotalDuration   int64       `json:"total_duration"`
	LoadDuration    int64       `json:"load_duration"`
	PromptEvalCount int         `json:"prompt_eval_count"`
	EvalCount       int         `json:"eval_count"`
	EvalDuration    int64       `json:"eval_duration"`
}

// GenerateResponse represents the structure of a generate response from Ollama
type GenerateResponse struct {
	Model           string `json:"model"`
	CreatedAt       string `json:"created_at"`
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	DoneReason      string `json:"done_reason,omitempty"`
	TotalDuration   int64  `json:"total_duration"`
	LoadDuration    int64  `json:"load_duration"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
	EvalDuration    int64  `json:"eval_duration"`
}

// EmbedResponse represents the structure of an embedding response from Ollama
type EmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"`
	TotalDuration   int64       `json:"total_duration"`
	LoadDuration    int64       `json:"load_duration"`
	PromptEvalCount int         `json:"prompt_eval_count"`
}

// // responseWriter is a custom response writer that captures the response body
// type responseWriter struct {
// 	http.ResponseWriter
// 	body *bytes.Buffer
// }

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

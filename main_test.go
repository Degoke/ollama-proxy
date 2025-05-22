package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestLoadConfig tests the configuration loading functionality
func TestLoadConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("OLLAMA_URL", "http://test-ollama:11434")
	os.Setenv("EXTERNAL_VALIDATION_URL", "http://test-validation:8080")
	os.Setenv("API_KEY_HEADER_NAME", "X-Test-API-Key")
	os.Setenv("PROXY_PORT", "9090")
	os.Setenv("EXTERNAL_SERVER_API_KEY", "test-server-key")

	// Load configuration
	loadConfig()

	// Verify configuration values
	if ollamaURL != "http://test-ollama:11434" {
		t.Errorf("Expected ollamaURL to be http://test-ollama:11434, got %s", ollamaURL)
	}
	if externalValidationURL != "http://test-validation:8080" {
		t.Errorf("Expected externalValidationURL to be http://test-validation:8080, got %s", externalValidationURL)
	}
	if apiKeyHeaderName != "X-Test-API-Key" {
		t.Errorf("Expected apiKeyHeaderName to be X-Test-API-Key, got %s", apiKeyHeaderName)
	}
	if proxyPort != "9090" {
		t.Errorf("Expected proxyPort to be 9090, got %s", proxyPort)
	}
	if externalServerAPIKey != "test-server-key" {
		t.Errorf("Expected externalServerAPIKey to be test-server-key, got %s", externalServerAPIKey)
	}
}

// TestProxyHandler tests the proxy handler functionality
func TestProxyHandler(t *testing.T) {
	// Create mock servers
	ollamaServer := mockOllamaServer(t)
	defer ollamaServer.Close()
	validationServer := mockValidationServer(t, true, false)
	defer validationServer.Close()
	metricsServer := mockMetricsServer(t)
	defer metricsServer.Close()

	// Set up test environment
	ollamaURL = ollamaServer.URL
	externalValidationURL = validationServer.URL
	externalMetricsURL = metricsServer.URL

	// Create test cases
	testCases := []struct {
		name           string
		apiKey         string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:           "Missing API Key",
			apiKey:         "",
			requestBody:    nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Valid Chat Request",
			apiKey: "test-api-key",
			requestBody: ChatRequest{
				Model: "llama2",
				Messages: []ChatMessage{
					{
						Role:    "user",
						Content: "Hello, how are you?",
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Invalid Request Body",
			apiKey: "test-api-key",
			requestBody: map[string]interface{}{
				"invalid": "body",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Rate Limited Request",
			apiKey: "test-api-key",
			requestBody: ChatRequest{
				Model: "llama2",
				Messages: []ChatMessage{
					{
						Role:    "user",
						Content: "Hello, how are you?",
					},
				},
			},
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test request
			var body []byte
			if tc.requestBody != nil {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest("POST", "/api/chat", bytes.NewBuffer(body))
			if tc.apiKey != "" {
				req.Header.Set(apiKeyHeaderName, tc.apiKey)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			proxyHandler(rr, req)

			// Check status code
			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
		})
	}
}

// TestGetModelFromRequest tests the model extraction from different request types
func TestGetModelFromRequest(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		requestBody   interface{}
		expectedModel string
	}{
		{
			name: "Chat Request",
			path: "/api/chat",
			requestBody: ChatRequest{
				Model: "llama2",
			},
			expectedModel: "llama2",
		},
		{
			name: "Generate Request",
			path: "/api/generate",
			requestBody: GenerateRequest{
				Model: "mistral",
			},
			expectedModel: "mistral",
		},
		{
			name: "Embed Request",
			path: "/api/embed",
			requestBody: EmbedRequest{
				Model: "nomic-embed",
			},
			expectedModel: "nomic-embed",
		},
		{
			name: "Create Request",
			path: "/api/create",
			requestBody: CreateRequest{
				Model: "custom-model",
			},
			expectedModel: "custom-model",
		},
		{
			name:          "Invalid JSON",
			path:          "/api/chat",
			requestBody:   []byte("invalid json"),
			expectedModel: "",
		},
		{
			name:          "Unknown Endpoint",
			path:          "/api/unknown",
			requestBody:   nil,
			expectedModel: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if tc.requestBody != nil {
				if b, ok := tc.requestBody.([]byte); ok {
					body = b
				} else {
					body, _ = json.Marshal(tc.requestBody)
				}
			}
			model := getModelFromRequest(tc.path, body)
			if model != tc.expectedModel {
				t.Errorf("Expected model %s, got %s", tc.expectedModel, model)
			}
		})
	}
}

// TestGetTokenCountsFromResponse tests token count extraction from responses
func TestGetTokenCountsFromResponse(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		responseBody   interface{}
		expectedInput  int
		expectedOutput int
	}{
		{
			name: "Chat Response",
			path: "/api/chat",
			responseBody: ChatResponse{
				PromptEvalCount: 10,
				EvalCount:       20,
			},
			expectedInput:  10,
			expectedOutput: 20,
		},
		{
			name: "Generate Response",
			path: "/api/generate",
			responseBody: GenerateResponse{
				PromptEvalCount: 15,
				EvalCount:       25,
			},
			expectedInput:  15,
			expectedOutput: 25,
		},
		{
			name: "Embed Response",
			path: "/api/embed",
			responseBody: EmbedResponse{
				PromptEvalCount: 5,
			},
			expectedInput:  5,
			expectedOutput: 0,
		},
		{
			name:           "Invalid JSON",
			path:           "/api/chat",
			responseBody:   []byte("invalid json"),
			expectedInput:  0,
			expectedOutput: 0,
		},
		{
			name:           "Unknown Endpoint",
			path:           "/api/unknown",
			responseBody:   nil,
			expectedInput:  0,
			expectedOutput: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if tc.responseBody != nil {
				if b, ok := tc.responseBody.([]byte); ok {
					body = b
				} else {
					body, _ = json.Marshal(tc.responseBody)
				}
			}
			inputTokens, outputTokens := getTokenCountsFromResponse(tc.path, body)
			if inputTokens != tc.expectedInput {
				t.Errorf("Expected input tokens %d, got %d", tc.expectedInput, inputTokens)
			}
			if outputTokens != tc.expectedOutput {
				t.Errorf("Expected output tokens %d, got %d", tc.expectedOutput, outputTokens)
			}
		})
	}
}

// TestResponseWriter tests the custom response writer
func TestResponseWriter(t *testing.T) {
	// Create a test response writer
	rr := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rr,
		body:           &bytes.Buffer{},
	}

	// Test writing to the response
	testData := []byte("test response")
	_, err := rw.Write(testData)
	if err != nil {
		t.Errorf("Error writing to response: %v", err)
	}

	// Check if data was written to both the response and the buffer
	if rr.Body.String() != "test response" {
		t.Errorf("Expected response body to be 'test response', got '%s'", rr.Body.String())
	}
	if rw.body.String() != "test response" {
		t.Errorf("Expected buffer to contain 'test response', got '%s'", rw.body.String())
	}

	// Test writing empty data
	_, err = rw.Write(nil)
	if err != nil {
		t.Errorf("Error writing empty data: %v", err)
	}
}

// TestGetSecureHTTPClient tests the secure HTTP client creation
func TestGetSecureHTTPClient(t *testing.T) {
	// Test with default settings
	client := getSecureHTTPClient()
	if client == nil {
		t.Error("Expected non-nil HTTP client")
	}
}

// TestValidateRequest tests the request validation functionality
func TestValidateRequest(t *testing.T) {
	// Create test server for validation endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate validation response
		response := ValidationResponse{
			Valid:       true,
			RateLimited: false,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set validation URL to test server
	externalValidationURL = server.URL

	// Test valid request
	details := RequestDetails{
		APIKey:    "test-key",
		IPAddress: "127.0.0.1",
		Model:     "llama2",
	}
	if !validateRequest(details) {
		t.Error("Expected request to be valid")
	}

	// Test invalid request (simulate validation server error)
	server.Close()
	if validateRequest(details) {
		t.Error("Expected request to be invalid when validation server is down")
	}

	// Test rate limited request
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ValidationResponse{
			Valid:       true,
			RateLimited: true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	externalValidationURL = server.URL
	if validateRequest(details) {
		t.Error("Expected request to be invalid when rate limited")
	}
}

// TestSendMetrics tests the metrics sending functionality
func TestSendMetrics(t *testing.T) {
	// Create test server for metrics endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify metrics data
		var metrics MetricsData
		json.NewDecoder(r.Body).Decode(&metrics)
		if metrics.APIKey != "test-key" || metrics.Model != "llama2" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set metrics URL to test server
	externalMetricsURL = server.URL

	// Test sending metrics
	metrics := MetricsData{
		APIKey:            "test-key",
		Model:             "llama2",
		InputTokenLength:  10,
		OutputTokenLength: 20,
		RequestDurationMs: 100,
		Endpoint:          "/api/chat",
	}
	sendMetrics(metrics)

	// Test sending metrics with server down
	server.Close()
	sendMetrics(metrics) // Should not panic

	// Test sending metrics with invalid data
	metrics.APIKey = ""
	sendMetrics(metrics) // Should not panic
}

// TestValidateExternalServices tests the external service validation functionality
func TestValidateExternalServices(t *testing.T) {
	// Create mock servers
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected path /api/tags, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ollamaServer.Close()

	validationServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != externalServerAPIKey {
			t.Errorf("Expected X-API-Key header, got %s", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer validationServer.Close()

	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != externalServerAPIKey {
			t.Errorf("Expected X-API-Key header, got %s", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer metricsServer.Close()

	// Set up test environment
	ollamaURL = ollamaServer.URL
	externalValidationURL = validationServer.URL
	externalMetricsURL = metricsServer.URL
	externalServerAPIKey = "test-api-key"

	// Test successful validation
	if err := validateExternalServices(); err != nil {
		t.Errorf("Expected successful validation, got error: %v", err)
	}

	// Test Ollama service failure
	ollamaServer.Close()
	if err := validateExternalServices(); err == nil {
		t.Error("Expected validation error for Ollama service")
	}

	// Test validation service failure
	ollamaServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ollamaServer.Close()
	ollamaURL = ollamaServer.URL
	validationServer.Close()
	if err := validateExternalServices(); err == nil {
		t.Error("Expected validation error for validation service")
	}

	// Test metrics service failure
	validationServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer validationServer.Close()
	externalValidationURL = validationServer.URL
	metricsServer.Close()
	if err := validateExternalServices(); err == nil {
		t.Error("Expected validation error for metrics service")
	}
}

// TestValidateOllamaService tests the Ollama service validation
func TestValidateOllamaService(t *testing.T) {
	// Test successful validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected path /api/tags, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ollamaURL = server.URL
	if err := validateOllamaService(); err != nil {
		t.Errorf("Expected successful validation, got error: %v", err)
	}

	// Test server error
	server.Close()
	if err := validateOllamaService(); err == nil {
		t.Error("Expected validation error")
	}

	// Test non-OK status
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	ollamaURL = server.URL
	if err := validateOllamaService(); err == nil {
		t.Error("Expected validation error for non-OK status")
	}
}

// TestValidateExternalValidationService tests the external validation service validation
func TestValidateExternalValidationService(t *testing.T) {
	// Test successful validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != externalServerAPIKey {
			t.Errorf("Expected X-API-Key header, got %s", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	externalValidationURL = server.URL
	externalServerAPIKey = "test-api-key"
	if err := validateExternalValidationService(); err != nil {
		t.Errorf("Expected successful validation, got error: %v", err)
	}

	// Test server error
	server.Close()
	if err := validateExternalValidationService(); err == nil {
		t.Error("Expected validation error")
	}

	// Test unauthorized error
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	externalValidationURL = server.URL
	if err := validateExternalValidationService(); err == nil {
		t.Error("Expected validation error for unauthorized status")
	}
}

// TestValidateExternalMetricsService tests the external metrics service validation
func TestValidateExternalMetricsService(t *testing.T) {
	// Test successful validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != externalServerAPIKey {
			t.Errorf("Expected X-API-Key header, got %s", r.Header.Get("X-API-Key"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	externalMetricsURL = server.URL
	externalServerAPIKey = "test-api-key"
	if err := validateExternalMetricsService(); err != nil {
		t.Errorf("Expected successful validation, got error: %v", err)
	}

	// Test server error
	server.Close()
	if err := validateExternalMetricsService(); err == nil {
		t.Error("Expected validation error")
	}

	// Test unauthorized error
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	externalMetricsURL = server.URL
	if err := validateExternalMetricsService(); err == nil {
		t.Error("Expected validation error for unauthorized status")
	}
}

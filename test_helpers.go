package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockOllamaServer creates a test server that simulates Ollama's behavior
func mockOllamaServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Handle different endpoints
		switch r.URL.Path {
		case "/api/chat":
			response := ChatResponse{
				Model:           "llama2",
				CreatedAt:       "2024-01-01T00:00:00Z",
				Message:         ChatMessage{Role: "assistant", Content: "Hello! How can I help you?"},
				Done:            true,
				PromptEvalCount: 10,
				EvalCount:       20,
			}
			json.NewEncoder(w).Encode(response)

		case "/api/generate":
			response := GenerateResponse{
				Model:           "mistral",
				CreatedAt:       "2024-01-01T00:00:00Z",
				Response:        "Generated response",
				Done:            true,
				PromptEvalCount: 15,
				EvalCount:       25,
			}
			json.NewEncoder(w).Encode(response)

		case "/api/embed":
			response := EmbedResponse{
				Model:           "nomic-embed",
				Embeddings:      [][]float32{{0.1, 0.2, 0.3}},
				PromptEvalCount: 5,
			}
			json.NewEncoder(w).Encode(response)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// mockValidationServer creates a test server that simulates the validation service
func mockValidationServer(t *testing.T, valid bool, rateLimited bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var details RequestDetails
		if err := json.NewDecoder(r.Body).Decode(&details); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Send validation response
		response := ValidationResponse{
			Valid:       valid,
			RateLimited: rateLimited,
		}
		json.NewEncoder(w).Encode(response)
	}))
}

// mockMetricsServer creates a test server that simulates the metrics service
func mockMetricsServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var metrics MetricsData
		if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
			t.Errorf("Error decoding request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify required fields
		if metrics.APIKey == "" || metrics.Model == "" {
			t.Error("Missing required fields in metrics data")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
}

// createTestRequest creates a test HTTP request with the given parameters
func createTestRequest(t *testing.T, method, path string, body interface{}, apiKey string) *http.Request {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Error marshaling request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set(apiKeyHeaderName, apiKey)
	}

	return req
}

// assertResponseStatus checks if the response status matches the expected status
func assertResponseStatus(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) {
	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
	}
}

// assertResponseBody checks if the response body matches the expected body
func assertResponseBody(t *testing.T, rr *httptest.ResponseRecorder, expectedBody interface{}) {
	var response, expected []byte
	var err error

	// Marshal expected body if it's not already a byte slice
	if expectedBody != nil {
		expected, err = json.Marshal(expectedBody)
		if err != nil {
			t.Fatalf("Error marshaling expected body: %v", err)
		}
	}

	// Get response body
	response = rr.Body.Bytes()

	// Compare bodies
	if !bytes.Equal(response, expected) {
		t.Errorf("Expected body %s, got %s", expected, response)
	}
}

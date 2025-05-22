package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// ValidationResponse represents the response from the validation service
type ValidationResponse struct {
	Valid       bool `json:"valid"`
	RateLimited bool `json:"rateLimited"`
}

// RequestDetails represents the request details sent to the validation service
type RequestDetails struct {
	APIKey    string            `json:"apiKey"`
	IPAddress string            `json:"ipAddress"`
	UserAgent string            `json:"userAgent"`
	Headers   map[string]string `json:"headers"`
	Endpoint  string            `json:"endpoint"`
	Model     string            `json:"model"`
}

// MetricsData represents the metrics data sent to the metrics service
type MetricsData struct {
	APIKey            string `json:"apiKey"`
	Model             string `json:"model"`
	InputTokenLength  int    `json:"inputTokenLength"`
	OutputTokenLength int    `json:"outputTokenLength"`
	RequestDurationMs int64  `json:"requestDurationMs"`
	Endpoint          string `json:"endpoint"`
}

var (
	mainAPIKey        = "main-api-key"
	validAPIKey       = "test-api-key"
	rateLimitedAPIKey = "rate-limited-key"
)

func startMockService() {
	// Validation endpoint handler
	http.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		// Check API key
		if r.Header.Get("X-API-Key") != mainAPIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Handle GET request (health check)
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle POST request (validation)
		if r.Method == http.MethodPost {
			var details RequestDetails
			if err := json.NewDecoder(r.Body).Decode(&details); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			// Simple validation logic
			response := ValidationResponse{
				Valid:       false,
				RateLimited: false,
			}

			if details.APIKey == validAPIKey {
				response.Valid = true
			}

			// Simulate rate limiting for specific API keys
			if details.APIKey == rateLimitedAPIKey {
				response.RateLimited = true
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Metrics endpoint handler
	http.HandleFunc("/log_metrics", func(w http.ResponseWriter, r *http.Request) {
		// Check API key
		if r.Header.Get("X-API-Key") != mainAPIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Handle GET request (health check)
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle POST request (metrics)
		if r.Method == http.MethodPost {
			var metrics MetricsData
			if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			// Log the metrics (in a real service, this would be stored in a database)
			log.Printf("Received metrics: %+v", metrics)
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Start the server
	port := 3000
	log.Printf("Starting mock external service on port %d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("Failed to start mock service: %v", err)
	}
}

func main() {
	startMockService()
}

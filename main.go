package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"ollama-proxy/logger"
)

// Configuration variables
var (
	ollamaURL             string
	externalValidationURL string
	externalMetricsURL    string
	apiKeyHeaderName      string
	proxyPort             string
	reverseProxy          *httputil.ReverseProxy
	proxyOnce             sync.Once

	// Security configuration
	externalServerAPIKey string
	externalServerCert   string
	skipTLSVerify        bool
)

type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func main() {
	// Load .env in development
	if os.Getenv("GO_ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			logger.Warning("No .env file found or error loading .env", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Load configuration from environment variables
	loadConfig()

	// Validate external services
	if err := validateExternalServices(); err != nil {
		logger.Error("Failed to validate external services", err, nil)
		os.Exit(1)
	}

	// Set up HTTP server
	http.HandleFunc("/", proxyHandler)

	// Start server
	logger.Info("Starting Ollama proxy server", map[string]interface{}{
		"port": proxyPort,
	})
	if err := http.ListenAndServe(":"+proxyPort, nil); err != nil {
		logger.Error("Failed to start server", err, nil)
		os.Exit(1)
	}
}

func loadConfig() {
	ollamaURL = getEnvOrDefault("OLLAMA_URL", "http://localhost:11434")
	externalValidationURL = getEnvOrDefault("EXTERNAL_VALIDATION_URL", "http://external-server.com/validate")
	externalMetricsURL = getEnvOrDefault("EXTERNAL_METRICS_URL", "http://external-server.com/log_metrics")
	apiKeyHeaderName = getEnvOrDefault("API_KEY_HEADER_NAME", "X-API-Key")
	proxyPort = getEnvOrDefault("PROXY_PORT", "8080")

	// Load security configuration
	externalServerAPIKey = getEnvOrDefault("EXTERNAL_SERVER_API_KEY", "")
	externalServerCert = getEnvOrDefault("EXTERNAL_SERVER_CERT", "")
	skipTLSVerify = getEnvOrDefault("SKIP_TLS_VERIFY", "false") == "true"
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getReverseProxy() *httputil.ReverseProxy {
	proxyOnce.Do(func() {
		targetURL, err := url.Parse(ollamaURL)
		if err != nil {
			log.Fatalf("Failed to parse Ollama URL: %v", err)
		}

		reverseProxy = &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = targetURL.Scheme
				req.URL.Host = targetURL.Host
				req.URL.Path = singleJoiningSlash(targetURL.Path, req.URL.Path)
				if targetURL.RawQuery == "" || req.URL.RawQuery == "" {
					req.URL.RawQuery = targetURL.RawQuery + req.URL.RawQuery
				} else {
					req.URL.RawQuery = targetURL.RawQuery + "&" + req.URL.RawQuery
				}
			},
		}
	})
	return reverseProxy
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	fields := map[string]interface{}{
		"user_agent": r.Header.Get("User-Agent"),
		"endpoint":   r.URL.Path,
	}

	// Extract API key
	apiKey := r.Header.Get(apiKeyHeaderName)
	if apiKey == "" {
		logger.Warning("Unauthorized: Missing API key", fields)
		http.Error(w, "Unauthorized: Missing API key", http.StatusUnauthorized)
		return
	}
	fields["api_key"] = apiKey

	// Extract request details
	details := RequestDetails{
		APIKey:    apiKey,
		IPAddress: r.RemoteAddr,
		UserAgent: r.Header.Get("User-Agent"),
		Headers:   make(map[string]string),
		Endpoint:  r.URL.Path,
	}

	// Copy headers
	for k, v := range r.Header {
		details.Headers[k] = v[0]
	}

	// Parse request body to get model and estimate token length
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Error reading request body", err, fields)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Get model from request based on endpoint
	details.Model = getModelFromRequest(r.URL.Path, bodyBytes)
	fields["model"] = details.Model

	// Validate request
	if !validateRequest(details) {
		logger.Warning("Unauthorized: Invalid request", fields)
		http.Error(w, "Unauthorized: Invalid request", http.StatusUnauthorized)
		return
	}

	// Create response writer to capture the response
	responseWriter := &responseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
	}

	// Proxy the request
	proxy := getReverseProxy()
	proxy.ServeHTTP(responseWriter, r)

	// Calculate metrics
	duration := time.Since(startTime)

	// Get token counts from Ollama response
	inputTokens, outputTokens := getTokenCountsFromResponse(r.URL.Path, responseWriter.body.Bytes())
	fields["input_tokens"] = inputTokens
	fields["output_tokens"] = outputTokens
	fields["duration_ms"] = duration.Milliseconds()

	// Log the request
	logger.RequestLog(r.Method, r.URL.Path, r.RemoteAddr, responseWriter.statusCode, duration, fields)

	// Send metrics asynchronously
	go sendMetrics(MetricsData{
		APIKey:            apiKey,
		Model:             details.Model,
		InputTokenLength:  inputTokens,
		OutputTokenLength: outputTokens,
		RequestDurationMs: duration.Milliseconds(),
		Endpoint:          details.Endpoint,
	})
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func getModelFromRequest(path string, body []byte) string {
	switch {
	case strings.HasSuffix(path, "/api/chat"):
		var chatReq ChatRequest
		if err := json.Unmarshal(body, &chatReq); err == nil {
			return chatReq.Model
		}
	case strings.HasSuffix(path, "/api/generate"):
		var genReq GenerateRequest
		if err := json.Unmarshal(body, &genReq); err == nil {
			return genReq.Model
		}
	case strings.HasSuffix(path, "/api/embed"):
		var embedReq EmbedRequest
		if err := json.Unmarshal(body, &embedReq); err == nil {
			return embedReq.Model
		}
	case strings.HasSuffix(path, "/api/create"):
		var createReq CreateRequest
		if err := json.Unmarshal(body, &createReq); err == nil {
			return createReq.Model
		}
	}
	return ""
}

func getTokenCountsFromResponse(path string, responseBody []byte) (int, int) {
	var inputTokens, outputTokens int

	switch {
	case strings.HasSuffix(path, "/api/chat"):
		var chatResp ChatResponse
		if err := json.Unmarshal(responseBody, &chatResp); err == nil {
			inputTokens = chatResp.PromptEvalCount
			outputTokens = chatResp.EvalCount
		}
	case strings.HasSuffix(path, "/api/generate"):
		var genResp GenerateResponse
		if err := json.Unmarshal(responseBody, &genResp); err == nil {
			inputTokens = genResp.PromptEvalCount
			outputTokens = genResp.EvalCount
		}
	case strings.HasSuffix(path, "/api/embed"):
		var embedResp EmbedResponse
		if err := json.Unmarshal(responseBody, &embedResp); err == nil {
			inputTokens = embedResp.PromptEvalCount
			// Embeddings don't have output tokens in the same way
			outputTokens = 0
		}
	}

	return inputTokens, outputTokens
}

func getSecureHTTPClient() *http.Client {
	// Create a custom transport with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerify,
		},
	}

	// If a custom certificate is provided, load it
	if externalServerCert != "" {
		cert, err := tls.LoadX509KeyPair(externalServerCert, externalServerCert)
		if err != nil {
			log.Printf("Warning: Failed to load certificate: %v", err)
		} else {
			transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second, // Add timeout for external requests
	}
}

func validateRequest(details RequestDetails) bool {
	jsonData, err := json.Marshal(details)
	if err != nil {
		logger.Error("Error marshaling validation request", err, map[string]interface{}{
			"api_key":  details.APIKey,
			"endpoint": details.Endpoint,
		})
		return false
	}

	// Create request with authentication
	req, err := http.NewRequest("POST", externalValidationURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Error creating validation request", err, map[string]interface{}{
			"api_key":  details.APIKey,
			"endpoint": details.Endpoint,
		})
		return false
	}

	// Add security headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", externalServerAPIKey)
	req.Header.Set("X-Request-ID", fmt.Sprintf("%d", time.Now().UnixNano()))

	// Use secure client
	client := getSecureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error calling validation server", err, map[string]interface{}{
			"api_key":  details.APIKey,
			"endpoint": details.Endpoint,
		})
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warning("Validation server returned non-OK status", map[string]interface{}{
			"api_key":     details.APIKey,
			"endpoint":    details.Endpoint,
			"status_code": resp.StatusCode,
		})
		return false
	}

	var validationResp ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResp); err != nil {
		logger.Error("Error decoding validation response", err, map[string]interface{}{
			"api_key":  details.APIKey,
			"endpoint": details.Endpoint,
		})
		return false
	}

	return validationResp.Valid && !validationResp.RateLimited
}

func sendMetrics(metrics MetricsData) {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		logger.Error("Error marshaling metrics", err, map[string]interface{}{
			"api_key":  metrics.APIKey,
			"model":    metrics.Model,
			"endpoint": metrics.Endpoint,
		})
		return
	}

	// Create request with authentication
	req, err := http.NewRequest("POST", externalMetricsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Error creating metrics request", err, map[string]interface{}{
			"api_key":  metrics.APIKey,
			"model":    metrics.Model,
			"endpoint": metrics.Endpoint,
		})
		return
	}

	// Add security headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", externalServerAPIKey)
	req.Header.Set("X-Request-ID", fmt.Sprintf("%d", time.Now().UnixNano()))

	// Use secure client
	client := getSecureHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error sending metrics", err, map[string]interface{}{
			"api_key":  metrics.APIKey,
			"model":    metrics.Model,
			"endpoint": metrics.Endpoint,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warning("Metrics server returned non-OK status", map[string]interface{}{
			"api_key":     metrics.APIKey,
			"model":       metrics.Model,
			"endpoint":    metrics.Endpoint,
			"status_code": resp.StatusCode,
		})
	}
}

// validateExternalServices checks if all required external services are accessible
func validateExternalServices() error {
	// Validate Ollama service
	if err := validateOllamaService(); err != nil {
		return fmt.Errorf("Ollama service validation failed: %v", err)
	}

	// Validate external validation service
	if err := validateExternalValidationService(); err != nil {
		return fmt.Errorf("External validation service validation failed: %v", err)
	}

	// Validate external metrics service
	if err := validateExternalMetricsService(); err != nil {
		return fmt.Errorf("External metrics service validation failed: %v", err)
	}

	return nil
}

// validateOllamaService checks if the Ollama service is accessible
func validateOllamaService() error {
	client := getSecureHTTPClient()
	resp, err := client.Get(ollamaURL + "/api/tags")
	if err != nil {
		logger.Error("Failed to connect to Ollama service", err, nil)
		return fmt.Errorf("failed to connect to Ollama service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warning("Ollama service returned non-OK status", map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		return fmt.Errorf("Ollama service returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

// validateExternalValidationService checks if the external validation service is accessible
func validateExternalValidationService() error {
	client := getSecureHTTPClient()
	req, err := http.NewRequest("GET", externalValidationURL, nil)
	if err != nil {
		logger.Error("Failed to create validation request", err, nil)
		return fmt.Errorf("failed to create validation request: %v", err)
	}

	// Add security headers
	req.Header.Set("X-API-Key", externalServerAPIKey)
	req.Header.Set("X-Request-ID", fmt.Sprintf("%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to connect to validation service", err, nil)
		return fmt.Errorf("failed to connect to validation service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warning("Validation service returned non-OK status", map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		return fmt.Errorf("validation service returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

// validateExternalMetricsService checks if the external metrics service is accessible
func validateExternalMetricsService() error {
	client := getSecureHTTPClient()
	req, err := http.NewRequest("GET", externalMetricsURL, nil)
	if err != nil {
		logger.Error("Failed to create metrics request", err, nil)
		return fmt.Errorf("failed to create metrics request: %v", err)
	}

	// Add security headers
	req.Header.Set("X-API-Key", externalServerAPIKey)
	req.Header.Set("X-Request-ID", fmt.Sprintf("%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to connect to metrics service", err, nil)
		return fmt.Errorf("failed to connect to metrics service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warning("Metrics service returned non-OK status", map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		return fmt.Errorf("metrics service returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

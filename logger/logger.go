package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	DEBUG   LogLevel = "DEBUG"
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var (
	// Default logger instance
	defaultLogger = log.New(os.Stdout, "", 0)
)

// Log writes a structured log entry
func Log(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		defaultLogger.Printf("Error marshaling log entry: %v", err)
		return
	}

	defaultLogger.Println(string(jsonBytes))
}

// Debug logs a debug message
func Debug(message string, fields map[string]interface{}) {
	Log(DEBUG, message, fields)
}

// Info logs an info message
func Info(message string, fields map[string]interface{}) {
	Log(INFO, message, fields)
}

// Warning logs a warning message
func Warning(message string, fields map[string]interface{}) {
	Log(WARNING, message, fields)
}

// Error logs an error message
func Error(message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	Log(ERROR, message, fields)
}

// RequestLog logs information about an HTTP request
func RequestLog(method, path, remoteAddr string, statusCode int, duration time.Duration, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["method"] = method
	fields["path"] = path
	fields["remote_addr"] = remoteAddr
	fields["status_code"] = statusCode
	fields["duration_ms"] = duration.Milliseconds()

	level := INFO
	if statusCode >= 400 {
		level = ERROR
	} else if statusCode >= 300 {
		level = WARNING
	}

	Log(level, fmt.Sprintf("%s %s %d", method, path, statusCode), fields)
}

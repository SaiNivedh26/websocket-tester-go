package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
	"sort"
)

// isValidJSON checks if a string is valid JSON
func isValidJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// validateWebSocketURL validates and normalizes a WebSocket URL
func validateWebSocketURL(urlStr string) (string, error) {
	// Add ws:// prefix if not present
	if !strings.HasPrefix(urlStr, "ws://") && !strings.HasPrefix(urlStr, "wss://") {
		urlStr = "ws://" + urlStr
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Validate scheme
	if parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss" {
		return "", fmt.Errorf("unsupported scheme: %s (use ws:// or wss://)", parsedURL.Scheme)
	}

	// Validate host
	if parsedURL.Host == "" {
		return "", fmt.Errorf("missing host in URL")
	}

	return urlStr, nil
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// formatBytes formats bytes in a human-readable way
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// calculatePercentile calculates the nth percentile from a slice of durations
func calculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Sort latencies (in-place)
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	// Calculate index for percentile
	index := (percentile * len(latencies)) / 100
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

// validateTestOptions validates the test configuration options
func validateTestOptions(opts *TestOptions) error {
	// Validate URL
	if _, err := validateWebSocketURL(opts.URL); err != nil {
		return fmt.Errorf("URL validation failed: %v", err)
	}

	// Validate duration
	if _, err := time.ParseDuration(opts.Duration); err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}

	// Validate connections
	if opts.Connections <= 0 {
		return fmt.Errorf("connections must be greater than 0")
	}

	// Validate loop count
	if opts.Loop <= 0 {
		return fmt.Errorf("loop count must be greater than 0")
	}

	// Validate message (check if it's valid JSON if it looks like JSON)
	if strings.TrimSpace(opts.Message) == "" {
		return fmt.Errorf("message cannot be empty")
	}

	// If message looks like JSON, validate it
	if strings.HasPrefix(strings.TrimSpace(opts.Message), "{") || 
	   strings.HasPrefix(strings.TrimSpace(opts.Message), "[") {
		if !isValidJSON(opts.Message) {
			return fmt.Errorf("message appears to be JSON but is not valid: %s", opts.Message)
		}
	}

	return nil
}

// sanitizeMessage ensures the message is safe to display
func sanitizeMessage(message string, maxLength int) string {
	if len(message) <= maxLength {
		return message
	}
	return message[:maxLength] + "..."
} 
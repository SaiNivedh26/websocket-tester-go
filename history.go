package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TestHistoryEntry represents a single test run entry
type TestHistoryEntry struct {
	ID             int            `json:"id"`
	Timestamp      time.Time      `json:"timestamp"`
	URL            string         `json:"url"`
	Duration       string         `json:"duration"`
	ActualDuration float64        `json:"actual_duration"` // in seconds
	Connections    int            `json:"connections"`
	Message        string         `json:"message"`
	LoopCount      int            `json:"loop_count"`
	TotalRequests  int64          `json:"total_requests"`
	SuccessfulReqs int64          `json:"successful_requests"`
	FailedReqs     int64          `json:"failed_requests"`
	SuccessRate    float64        `json:"success_rate"`
	AvgLatency     float64        `json:"avg_latency_ms"`
	P50Latency     float64        `json:"p50_latency_ms"`
	RequestsPerSec float64        `json:"requests_per_sec"`
	Throughput     float64        `json:"throughput_bytes_sec"`
	BytesSent      int64          `json:"bytes_sent"`
	BytesReceived  int64          `json:"bytes_received"`
	ErrorCounts    map[string]int `json:"error_counts"`
}

// TestHistory manages the collection of test history entries
type TestHistory struct {
	Entries []TestHistoryEntry `json:"entries"`
}

// getHistoryFilePath returns the path to the history file in temp directory
func getHistoryFilePath() string {
	tempDir := getTempDirPath()
	return filepath.Join(tempDir, "ws-load-history.json")
}

// getTempDirPath returns the path to the temp directory for saving charts and history
func getTempDirPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "temp"
	}
	tempDir := filepath.Join(homeDir, "temp")

	// Create temp directory if it doesn't exist
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.MkdirAll(tempDir, 0755)
	}

	return tempDir
}

// loadHistory loads the test history from file
func loadHistory() (*TestHistory, error) {
	historyPath := getHistoryFilePath()

	// Check if file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return &TestHistory{Entries: []TestHistoryEntry{}}, nil
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history file: %v", err)
	}

	var history TestHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %v", err)
	}

	return &history, nil
}

// saveHistory saves the test history to file
func (th *TestHistory) saveHistory() error {
	historyPath := getHistoryFilePath()

	data, err := json.MarshalIndent(th, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %v", err)
	}

	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %v", err)
	}

	return nil
}

// addEntry adds a new test result to the history
func (th *TestHistory) addEntry(lt *LoadTest) error {
	lt.results.mu.RLock()
	defer lt.results.mu.RUnlock()

	// Calculate metrics
	duration := lt.results.EndTime.Sub(lt.results.StartTime)
	totalRequests := lt.results.TotalRequests
	successfulReqs := lt.results.SuccessfulReqs
	failedReqs := lt.results.FailedReqs

	var avgLatency, p50Latency float64
	if successfulReqs > 0 {
		avgLatency = float64(lt.results.TotalLatency.Nanoseconds()) / float64(successfulReqs) / 1e6 // Convert to milliseconds
	}

	if len(lt.results.Latencies) > 0 {
		// Sort latencies for proper P50 calculation
		sortedLatencies := make([]time.Duration, len(lt.results.Latencies))
		copy(sortedLatencies, lt.results.Latencies)
		sort.Slice(sortedLatencies, func(i, j int) bool {
			return sortedLatencies[i] < sortedLatencies[j]
		})
		p50Latency = float64(sortedLatencies[len(sortedLatencies)/2].Nanoseconds()) / 1e6 // Convert to milliseconds
	}

	rps := float64(totalRequests) / duration.Seconds()
	throughput := float64(lt.results.BytesSent+lt.results.BytesReceived) / duration.Seconds()
	successRate := float64(successfulReqs) / float64(totalRequests) * 100

	// Generate new ID
	newID := 1
	if len(th.Entries) > 0 {
		newID = th.Entries[len(th.Entries)-1].ID + 1
	}

	entry := TestHistoryEntry{
		ID:             newID,
		Timestamp:      lt.results.StartTime,
		URL:            lt.opts.URL,
		Duration:       lt.opts.Duration,
		ActualDuration: duration.Seconds(),
		Connections:    lt.opts.Connections,
		Message:        lt.opts.Message,
		LoopCount:      lt.opts.Loop,
		TotalRequests:  totalRequests,
		SuccessfulReqs: successfulReqs,
		FailedReqs:     failedReqs,
		SuccessRate:    successRate,
		AvgLatency:     avgLatency,
		P50Latency:     p50Latency,
		RequestsPerSec: rps,
		Throughput:     throughput,
		BytesSent:      lt.results.BytesSent,
		BytesReceived:  lt.results.BytesReceived,
		ErrorCounts:    make(map[string]int),
	}

	// Copy error counts
	for k, v := range lt.results.ErrorCounts {
		entry.ErrorCounts[k] = v
	}

	th.Entries = append(th.Entries, entry)
	return th.saveHistory()
}

// getLastNEntries returns the last N entries from history
func (th *TestHistory) getLastNEntries(n int) []TestHistoryEntry {
	if n <= 0 || len(th.Entries) == 0 {
		return []TestHistoryEntry{}
	}

	start := len(th.Entries) - n
	if start < 0 {
		start = 0
	}

	return th.Entries[start:]
}

// printHistory displays the test history
func (th *TestHistory) printHistory(limit int) {
	if len(th.Entries) == 0 {
		fmt.Println("No test history found.")
		return
	}

	entries := th.getLastNEntries(limit)

	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    WebSocket Test History                   ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")

	for _, entry := range entries {
		fmt.Printf("Test #%d - %s\n", entry.ID, entry.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  URL:            %s\n", entry.URL)
		fmt.Printf("  Duration:       %s (actual: %.2fs)\n", entry.Duration, entry.ActualDuration)
		fmt.Printf("  Connections:    %d\n", entry.Connections)
		fmt.Printf("  Success Rate:   %.1f%% (%d/%d)\n", entry.SuccessRate, entry.SuccessfulReqs, entry.TotalRequests)
		fmt.Printf("  Requests/sec:   %.2f\n", entry.RequestsPerSec)
		fmt.Printf("  Avg Latency:    %.2fms\n", entry.AvgLatency)
		fmt.Printf("  Throughput:     %.2f bytes/sec\n", entry.Throughput)
		if len(entry.ErrorCounts) > 0 {
			fmt.Printf("  Errors:         ")
			for errorType, count := range entry.ErrorCounts {
				fmt.Printf("%s:%d ", errorType, count)
			}
			fmt.Printf("\n")
		}
		fmt.Printf("\n")
	}

	if len(th.Entries) > limit {
		fmt.Printf("Showing last %d of %d total tests\n", len(entries), len(th.Entries))
	}
}

// saveChartAsText saves the ASCII chart as a text file
func saveChartAsText(metric string, chartOutput string, timestamp time.Time) (string, error) {
	filename := fmt.Sprintf("ws-load-chart-%s-%s.txt",
		metric,
		timestamp.Format("2006-01-02-15-04-05"))
	filepath := filepath.Join(getTempDirPath(), filename)

	// Add header to the text file
	content := fmt.Sprintf("WebSocket Load Test - %s Trend Chart\n", metric)
	content += fmt.Sprintf("Generated on: %s\n", timestamp.Format("2006-01-02 15:04:05"))
	content += strings.Repeat("=", 60) + "\n\n"
	content += chartOutput

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to save chart text file: %v", err)
	}

	return filepath, nil
}

// clearHistory removes all history entries
func (th *TestHistory) clearHistory() error {
	th.Entries = []TestHistoryEntry{}
	return th.saveHistory()
}

// generateComparisonChart creates a simple ASCII chart comparing metrics and saves as PNG and text
func (th *TestHistory) generateComparisonChart(metric string, limit int) {
	if len(th.Entries) == 0 {
		fmt.Println("No test history found.")
		return
	}

	entries := th.getLastNEntries(limit)
	if len(entries) < 2 {
		fmt.Println("Need at least 2 test results for comparison.")
		return
	}

	timestamp := time.Now()

	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    %s Trend Chart                    ║\n", metric)
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")

	// Extract values based on metric type
	var values []float64
	var labels []string

	for _, entry := range entries {
		var value float64
		switch metric {
		case "success-rate":
			value = entry.SuccessRate
		case "requests-per-sec":
			value = entry.RequestsPerSec
		case "avg-latency":
			value = entry.AvgLatency
		case "throughput":
			value = entry.Throughput
		default:
			value = entry.SuccessRate
		}

		values = append(values, value)
		labels = append(labels, strconv.Itoa(entry.ID))
	}

	// Find min and max for scaling
	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Generate ASCII chart output
	var chartOutput strings.Builder

	// Generate ASCII chart
	chartHeight := 10

	chartOutput.WriteString("Test ID: ")
	for i, label := range labels {
		if i > 0 {
			chartOutput.WriteString("    ")
		}
		chartOutput.WriteString(fmt.Sprintf("%-4s", label))
	}
	chartOutput.WriteString("\n")

	for row := chartHeight; row >= 0; row-- {
		threshold := minVal + (maxVal-minVal)*float64(row)/float64(chartHeight)

		// Print Y-axis label
		chartOutput.WriteString(fmt.Sprintf("%7.2f |", threshold))

		// Print chart bars
		for i, value := range values {
			if i > 0 {
				chartOutput.WriteString("    ")
			}

			if value >= threshold {
				chartOutput.WriteString("████")
			} else {
				chartOutput.WriteString("    ")
			}
		}
		chartOutput.WriteString("\n")
	}

	// Print actual values
	chartOutput.WriteString("Values:  ")
	for i, value := range values {
		if i > 0 {
			chartOutput.WriteString("    ")
		}
		chartOutput.WriteString(fmt.Sprintf("%.2f", value))
	}
	chartOutput.WriteString("\n")

	// Display the chart
	fmt.Print(chartOutput.String())
	fmt.Printf("\n")

	// Save as text file
	textPath, err := saveChartAsText(metric, chartOutput.String(), timestamp)
	if err != nil {
		fmt.Printf("Warning: Could not save text chart: %v\n", err)
	} else {
		fmt.Printf("📄 Chart saved as text: %s\n", textPath)
	}

	fmt.Printf("\n💡 Chart saved to: %s\n", getTempDirPath())
}

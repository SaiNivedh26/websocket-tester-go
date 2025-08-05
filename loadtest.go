package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-metrics"
	"github.com/lxzan/gws"
	"github.com/schollz/progressbar/v3"
)

// Error categories for better error analysis
const (
	ErrorCategoryTimeout            = "timeout"
	ErrorCategoryConnectionRefused  = "connection_refused"
	ErrorCategoryUnexpectedResponse = "unexpected_response"
	ErrorCategoryInvalidData        = "invalid_data"
	ErrorCategoryAuthFailure        = "authentication_failure"
	ErrorCategoryNetworkError       = "network_error"
	ErrorCategoryProtocolError      = "protocol_error"
	ErrorCategoryResourceExhaustion = "resource_exhaustion"
	ErrorCategoryUnknown            = "unknown"
)

// ErrorCategoryInfo contains details about an error category
type ErrorCategoryInfo struct {
	Count       int
	Description string
	Examples    []string
}

// initializeErrorCategories creates and initializes error categories with descriptions
func initializeErrorCategories() map[string]*ErrorCategoryInfo {
	categories := make(map[string]*ErrorCategoryInfo)

	categories[ErrorCategoryTimeout] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Requests that timed out (handshake, read, write timeouts)",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryConnectionRefused] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Connection attempts rejected by the server",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryUnexpectedResponse] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Unexpected HTTP responses or WebSocket upgrade failures",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryInvalidData] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Invalid or malformed data received",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryAuthFailure] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Authentication or authorization failures",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryNetworkError] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Network-related errors (DNS, routing, etc.)",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryProtocolError] = &ErrorCategoryInfo{
		Count:       0,
		Description: "WebSocket protocol violations or errors",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryResourceExhaustion] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Resource exhaustion (too many connections, memory, etc.)",
		Examples:    make([]string, 0),
	}

	categories[ErrorCategoryUnknown] = &ErrorCategoryInfo{
		Count:       0,
		Description: "Uncategorized or unknown errors",
		Examples:    make([]string, 0),
	}

	return categories
}

// categorizeError determines the category of an error based on its message and type
func categorizeError(err error) string {
	if err == nil {
		return ErrorCategoryUnknown
	}

	errMsg := strings.ToLower(err.Error())

	// Timeout errors
	if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "handshake timeout") {
		return ErrorCategoryTimeout
	}

	// Connection refused errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connect: connection refused") ||
		strings.Contains(errMsg, "connectex: no connection could be made") ||
		strings.Contains(errMsg, "no connection could be made because the target machine actively refused") {
		return ErrorCategoryConnectionRefused
	}

	// Network errors
	if strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "network unreachable") ||
		strings.Contains(errMsg, "no route to host") ||
		strings.Contains(errMsg, "dns") {
		return ErrorCategoryNetworkError
	}

	// Authentication/Authorization errors
	if strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "403") {
		return ErrorCategoryAuthFailure
	}

	// Protocol errors
	if strings.Contains(errMsg, "bad handshake") ||
		strings.Contains(errMsg, "handshake error") ||
		strings.Contains(errMsg, "handshake") ||
		strings.Contains(errMsg, "websocket") ||
		strings.Contains(errMsg, "upgrade") ||
		strings.Contains(errMsg, "protocol") {
		return ErrorCategoryProtocolError
	}

	// Resource exhaustion
	if strings.Contains(errMsg, "too many") ||
		strings.Contains(errMsg, "resource temporarily unavailable") ||
		strings.Contains(errMsg, "out of memory") ||
		strings.Contains(errMsg, "no buffer space available") {
		return ErrorCategoryResourceExhaustion
	}

	// Unexpected response errors
	if strings.Contains(errMsg, "unexpected") ||
		strings.Contains(errMsg, "invalid response") ||
		strings.Contains(errMsg, "bad response") {
		return ErrorCategoryUnexpectedResponse
	}

	// Invalid data errors
	if strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "malformed") ||
		strings.Contains(errMsg, "parse") {
		return ErrorCategoryInvalidData
	}

	return ErrorCategoryUnknown
}

// LoadTest represents a WebSocket load test
type LoadTest struct {
	opts     *TestOptions
	metrics  *metrics.InmemSink
	results  *TestResults
	ctx      context.Context
	cancel   context.CancelFunc
	progress *progressbar.ProgressBar
	verbose  bool
}

// TestResults contains aggregated test results
type TestResults struct {
	mu               sync.RWMutex
	TotalRequests    int64
	SuccessfulReqs   int64
	FailedReqs       int64
	TotalLatency     time.Duration
	Latencies        []time.Duration
	PeakResponseTime time.Duration
	StartTime        time.Time
	EndTime          time.Time
	BytesSent        int64
	BytesReceived    int64
	ErrorCounts      map[string]int
	StatusCodeCount  map[int]int
	ErrorCategories  map[string]*ErrorCategoryInfo
}

// WebSocketEventHandler implements the gws.Event interface
type WebSocketEventHandler struct {
	connID int
	lt     *LoadTest
}

func (h *WebSocketEventHandler) OnOpen(socket *gws.Conn) {
	if h.lt.verbose {
		log.Printf("Connection %d opened", h.connID)
	}
}

func (h *WebSocketEventHandler) OnClose(socket *gws.Conn, err error) {
	if h.lt.verbose {
		log.Printf("Connection %d closed: %v", h.connID, err)
	}
}

func (h *WebSocketEventHandler) OnPing(socket *gws.Conn, payload []byte) {
	// Handle ping if needed
}

func (h *WebSocketEventHandler) OnPong(socket *gws.Conn, payload []byte) {
	// Handle pong if needed
}

func (h *WebSocketEventHandler) OnMessage(socket *gws.Conn, message *gws.Message) {
	// Record received bytes
	h.lt.results.mu.Lock()
	h.lt.results.BytesReceived += int64(message.Data.Len())
	h.lt.results.mu.Unlock()

	if h.lt.verbose {
		log.Printf("Connection %d received: %s", h.connID, message.Data.String())
	}
}

// NewLoadTest creates a new load test instance
func NewLoadTest(opts *TestOptions) *LoadTest {
	ctx, cancel := context.WithCancel(context.Background())

	return &LoadTest{
		opts:    opts,
		metrics: metrics.NewInmemSink(10*time.Second, 10*time.Minute),
		results: &TestResults{
			ErrorCounts:     make(map[string]int),
			StatusCodeCount: make(map[int]int),
			Latencies:       make([]time.Duration, 0),
			ErrorCategories: initializeErrorCategories(),
		},
		ctx:     ctx,
		cancel:  cancel,
		verbose: false,
	}
}

// Run executes the load test
func (lt *LoadTest) Run() error {
	// Parse duration
	duration, err := time.ParseDuration(lt.opts.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration format: %v", err)
	}

	// Set up progress bar
	lt.progress = progressbar.NewOptions64(
		int64(duration.Milliseconds()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][1/3][reset] Running WebSocket load test..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Record start time
	lt.results.StartTime = time.Now()

	// Start metrics collection
	go lt.collectMetrics()

	// Create connection pool
	var wg sync.WaitGroup
	connectionPool := make(chan struct{}, lt.opts.Connections)

	// Start connections
	for i := 0; i < lt.opts.Connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()
			lt.runConnection(connID, connectionPool)
		}(i)
	}

	// Wait for test duration
	select {
	case <-time.After(duration):
		lt.cancel()
	case <-lt.ctx.Done():
		// Test was cancelled
	}

	// Wait for all connections to finish
	wg.Wait()

	// Record end time
	lt.results.EndTime = time.Now()

	// Close progress bar
	lt.progress.Finish()

	// Print results
	lt.printResults()

	return nil
}

// runConnection handles a single WebSocket connection
func (lt *LoadTest) runConnection(connID int, pool chan struct{}) {
	// Acquire connection slot
	pool <- struct{}{}
	defer func() { <-pool }()

	// Create WebSocket client handler
	handler := &WebSocketEventHandler{
		connID: connID,
		lt:     lt,
	}

	// Create WebSocket client
	client, _, err := gws.NewClient(handler, &gws.ClientOption{
		Addr:             lt.opts.URL,
		HandshakeTimeout: 10 * time.Second,
	})
	if err != nil {
		lt.recordError(fmt.Sprintf("client_creation_failed_%d", connID), err)
		return
	}

	// Send messages in loop
	for i := 0; i < lt.opts.Loop; i++ {
		select {
		case <-lt.ctx.Done():
			return
		default:
			lt.sendMessage(client, connID, i)
		}
	}

	// Close connection gracefully
	client.WriteClose(1000, []byte("test completed"))
}

// sendMessage sends a single message and records metrics
func (lt *LoadTest) sendMessage(client *gws.Conn, connID, msgID int) {
	startTime := time.Now()

	// Send message
	err := client.WriteMessage(gws.OpcodeText, []byte(lt.opts.Message))
	if err != nil {
		lt.recordError(fmt.Sprintf("send_failed_%d_%d", connID, msgID), err)
		return
	}

	// Record metrics
	latency := time.Since(startTime)
	lt.results.mu.Lock()
	lt.results.TotalRequests++
	lt.results.SuccessfulReqs++
	lt.results.TotalLatency += latency
	lt.results.Latencies = append(lt.results.Latencies, latency)
	// Update peak response time if this latency is higher
	if latency > lt.results.PeakResponseTime {
		lt.results.PeakResponseTime = latency
	}
	lt.results.BytesSent += int64(len(lt.opts.Message))
	lt.results.mu.Unlock()

	// Update progress bar
	lt.progress.Add(1)
}

// recordError records an error occurrence
func (lt *LoadTest) recordError(errorType string, err error) {
	lt.results.mu.Lock()
	defer lt.results.mu.Unlock()

	lt.results.TotalRequests++
	lt.results.FailedReqs++
	lt.results.ErrorCounts[errorType]++

	// Categorize the error
	category := categorizeError(err)
	if categoryInfo, exists := lt.results.ErrorCategories[category]; exists {
		categoryInfo.Count++
		// Add example if we don't have too many already (limit to 3 examples per category)
		if len(categoryInfo.Examples) < 3 {
			categoryInfo.Examples = append(categoryInfo.Examples, err.Error())
		}
	}

	if lt.verbose {
		log.Printf("Error (%s) [%s]: %v", errorType, category, err)
	}
}

// collectMetrics periodically collects and reports metrics
func (lt *LoadTest) collectMetrics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lt.results.mu.RLock()
			rps := float64(lt.results.TotalRequests) / time.Since(lt.results.StartTime).Seconds()
			peakResponseTime := lt.results.PeakResponseTime

			// Collect error category counts for metrics
			errorCategoryMetrics := make(map[string]int)
			for category, info := range lt.results.ErrorCategories {
				errorCategoryMetrics[category] = info.Count
			}
			lt.results.mu.RUnlock()

			// Record RPS metric
			lt.metrics.SetGaugeWithLabels([]string{"rps"}, float32(rps), []metrics.Label{
				{Name: "test", Value: "websocket"},
			})

			// Record Peak Response Time metric (in milliseconds)
			lt.metrics.SetGaugeWithLabels([]string{"peak_response_time_ms"}, float32(peakResponseTime.Milliseconds()), []metrics.Label{
				{Name: "test", Value: "websocket"},
			})

			// Record error category metrics
			for category, count := range errorCategoryMetrics {
				lt.metrics.SetGaugeWithLabels([]string{"error_category_count"}, float32(count), []metrics.Label{
					{Name: "test", Value: "websocket"},
					{Name: "category", Value: category},
				})
			}
		case <-lt.ctx.Done():
			return
		}
	}
}

// printResults displays the final test results
func (lt *LoadTest) printResults() {
	lt.results.mu.RLock()
	defer lt.results.mu.RUnlock()

	duration := lt.results.EndTime.Sub(lt.results.StartTime)
	totalRequests := lt.results.TotalRequests
	successfulReqs := lt.results.SuccessfulReqs
	failedReqs := lt.results.FailedReqs

	var avgLatency time.Duration
	if successfulReqs > 0 {
		avgLatency = lt.results.TotalLatency / time.Duration(successfulReqs)
	}

	// Calculate P50 latency
	var p50Latency time.Duration
	if len(lt.results.Latencies) > 0 {
		// Simple P50 calculation (for production, use proper percentile calculation)
		p50Latency = lt.results.Latencies[len(lt.results.Latencies)/2]
	}

	rps := float64(totalRequests) / duration.Seconds()
	throughput := float64(lt.results.BytesSent+lt.results.BytesReceived) / duration.Seconds()

	fmt.Printf("\n\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    WebSocket Load Test Results              ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("Test Configuration:\n")
	fmt.Printf("  URL:         %s\n", lt.opts.URL)
	fmt.Printf("  Duration:    %s\n", duration)
	fmt.Printf("  Connections: %d\n", lt.opts.Connections)
	fmt.Printf("  Message:     %s\n", lt.opts.Message)
	fmt.Printf("  Loop Count:  %d\n", lt.opts.Loop)
	fmt.Printf("\n")
	fmt.Printf("Performance Metrics:\n")
	fmt.Printf("  Total Requests:     %d\n", totalRequests)
	fmt.Printf("  Successful:         %d (%.1f%%)\n", successfulReqs, float64(successfulReqs)/float64(totalRequests)*100)
	fmt.Printf("  Failed:             %d (%.1f%%)\n", failedReqs, float64(failedReqs)/float64(totalRequests)*100)
	fmt.Printf("  Requests/sec:       %.2f\n", rps)
	fmt.Printf("  Avg Latency:        %s\n", avgLatency)
	fmt.Printf("  P50 Latency:        %s\n", p50Latency)
	fmt.Printf("  Peak Response Time: %s\n", lt.results.PeakResponseTime)
	fmt.Printf("  Throughput:         %.2f bytes/sec\n", throughput)
	fmt.Printf("  Bytes Sent:         %d\n", lt.results.BytesSent)
	fmt.Printf("  Bytes Received:     %d\n", lt.results.BytesReceived)
	fmt.Printf("\n")

	if len(lt.results.ErrorCounts) > 0 {
		fmt.Printf("Error Summary:\n")
		for errorType, count := range lt.results.ErrorCounts {
			fmt.Printf("  %s: %d\n", errorType, count)
		}
		fmt.Printf("\n")

		// Print Error Categories
		fmt.Printf("Error Categories:\n")
		hasErrors := false
		for category, info := range lt.results.ErrorCategories {
			if info.Count > 0 {
				hasErrors = true
				fmt.Printf("  %s: %d (%.1f%%)\n",
					strings.Title(strings.ReplaceAll(category, "_", " ")),
					info.Count,
					float64(info.Count)/float64(failedReqs)*100)
				fmt.Printf("    └─ %s\n", info.Description)

				// Show examples if available
				if len(info.Examples) > 0 {
					fmt.Printf("    └─ Examples:\n")
					for i, example := range info.Examples {
						if len(example) > 80 {
							example = example[:77] + "..."
						}
						fmt.Printf("       %d. %s\n", i+1, example)
					}
				}
				fmt.Printf("\n")
			}
		}

		if !hasErrors {
			fmt.Printf("  No categorized errors found.\n\n")
		}
	}

	fmt.Printf("Test completed in %s\n", duration)
}

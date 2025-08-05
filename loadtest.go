package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-metrics"
	"github.com/lxzan/gws"
	"github.com/schollz/progressbar/v3"
)

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

	if lt.verbose {
		log.Printf("Error (%s): %v", errorType, err)
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
			lt.results.mu.RUnlock()

			// Record RPS metric
			lt.metrics.SetGaugeWithLabels([]string{"rps"}, float32(rps), []metrics.Label{
				{Name: "test", Value: "websocket"},
			})

			// Record Peak Response Time metric (in milliseconds)
			lt.metrics.SetGaugeWithLabels([]string{"peak_response_time_ms"}, float32(peakResponseTime.Milliseconds()), []metrics.Label{
				{Name: "test", Value: "websocket"},
			})
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
	}

	fmt.Printf("Test completed in %s\n", duration)
}

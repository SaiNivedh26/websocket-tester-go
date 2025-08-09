package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

// GlobalOptions contains options that apply to all commands
type GlobalOptions struct {
	Verbose bool `short:"v" long:"verbose" description:"Enable verbose output"`
}

// TestOptions contains options for the test command
type TestOptions struct {
	URL         string `short:"u" long:"url" description:"WebSocket endpoint URL (e.g., ws://echo.websocket.org)" required:"true"`
	Duration    string `short:"d" long:"duration" description:"Test duration (e.g., 10s, 5m, 1h)" default:"30s"`
	Connections int    `short:"c" long:"connections" description:"Number of concurrent connections" default:"10"`
	Message     string `short:"m" long:"message" description:"Message to send (string or JSON)" default:"Hello, WebSocket!"`
	Loop        int    `short:"l" long:"loop" description:"Number of times to send message per connection" default:"1"`
}

// ConfigOptions contains options for the config command
type ConfigOptions struct {
	Show bool `short:"s" long:"show" description:"Show current configuration"`
}

// HistoryOptions contains options for the history command
type HistoryOptions struct {
	Show  bool `short:"s" long:"show" description:"Show test history"`
	Limit int  `short:"l" long:"limit" description:"Number of recent tests to show" default:"10"`
	Clear bool `short:"c" long:"clear" description:"Clear all test history"`
}

// VisualizeOptions contains options for the visualize command
type VisualizeOptions struct {
	Metric string `short:"m" long:"metric" description:"Metric to visualize (success-rate, requests-per-sec, avg-latency, throughput)" default:"success-rate"`
	Limit  int    `short:"l" long:"limit" description:"Number of recent tests to include" default:"10"`
}

// Commands structure for the CLI
type Commands struct {
	Test      TestOptions      `command:"test" description:"Run a WebSocket load test"`
	Config    ConfigOptions    `command:"config" description:"Manage configuration"`
	History   HistoryOptions   `command:"history" description:"View test history"`
	Visualize VisualizeOptions `command:"visualize" description:"Visualize test metrics"`
}

func main() {
	var globalOpts GlobalOptions
	var commands Commands

	parser := flags.NewParser(&globalOpts, flags.Default)

	// Add commands individually
	testCmd, err := parser.AddCommand("test", "Run a WebSocket load test", "Execute a WebSocket load test with specified parameters", &commands.Test)
	if err != nil {
		log.Fatal("Failed to add test command:", err)
	}
	_ = testCmd

	configCmd, err := parser.AddCommand("config", "Manage configuration", "View and modify tool configuration", &commands.Config)
	if err != nil {
		log.Fatal("Failed to add config command:", err)
	}
	_ = configCmd

	historyCmd, err := parser.AddCommand("history", "View test history", "View historical test results and trends", &commands.History)
	if err != nil {
		log.Fatal("Failed to add history command:", err)
	}
	_ = historyCmd

	visualizeCmd, err := parser.AddCommand("visualize", "Visualize metrics", "Create charts and graphs from test history", &commands.Visualize)
	if err != nil {
		log.Fatal("Failed to add visualize command:", err)
	}
	_ = visualizeCmd

	// Set custom help template
	parser.LongDescription = `WebSocket Load Testing Tool

A high-performance, native Go CLI tool for WebSocket load testing that leverages Go's 
efficient goroutines and channels for highly concurrent load generation.

Features:
  • Real-time bi-directional WebSocket communication testing
  • High-performance load generation with minimal resource consumption
  • Comprehensive performance metrics (RPS, Latency, Throughput)
  • Human-first CLI design with progress indicators
  • Robust error handling and logging

Examples:
  ws-load test -u ws://echo.websocket.org -d 30s -c 50
  ws-load test --url ws://localhost:8080/ws --duration 5m --connections 100 --message '{"type":"ping"}'
  ws-load config --show
  ws-load history --limit 5
  ws-load visualize --metric requests-per-sec --limit 10`

	// Parse command line arguments
	_, parseErr := parser.Parse()
	if parseErr != nil {
		if flagsErr, ok := parseErr.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
		os.Exit(1)
	}

	// Handle commands
	if parser.Active == nil {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	switch parser.Active.Name {
	case "test":
		runTest(&commands.Test, &globalOpts)
	case "config":
		runConfig(&commands.Config, &globalOpts)
	case "history":
		runHistory(&commands.History, &globalOpts)
	case "visualize":
		runVisualize(&commands.Visualize, &globalOpts)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", parser.Active.Name)
		os.Exit(1)
	}
}

func runTest(opts *TestOptions, globalOpts *GlobalOptions) {
	// Validate test options
	if err := validateTestOptions(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	if globalOpts.Verbose {
		fmt.Printf("Starting WebSocket load test...\n")
		fmt.Printf("URL: %s\n", opts.URL)
		fmt.Printf("Duration: %s\n", opts.Duration)
		fmt.Printf("Connections: %d\n", opts.Connections)
		fmt.Printf("Message: %s\n", sanitizeMessage(opts.Message, 100))
		fmt.Printf("Loop count: %d\n", opts.Loop)
		fmt.Printf("Verbose mode: enabled\n")
	}

	// Create and run the load test
	test := NewLoadTest(opts)
	test.verbose = globalOpts.Verbose // Set verbose mode

	if err := test.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Test failed: %v\n", err)
		os.Exit(1)
	}

	// Save test results to history
	history, err := loadHistory()
	if err != nil {
		if globalOpts.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: Could not load history: %v\n", err)
		}
	} else {
		if err := history.addEntry(test); err != nil {
			if globalOpts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Could not save to history: %v\n", err)
			}
		} else if globalOpts.Verbose {
			fmt.Printf("Test results saved to history.\n")
		}
	}
}

func runConfig(opts *ConfigOptions, globalOpts *GlobalOptions) {
	if opts.Show {
		fmt.Println("Current Configuration:")
		fmt.Println("  Default duration: 30s")
		fmt.Println("  Default connections: 10")
		fmt.Println("  Default message: Hello, WebSocket!")
		fmt.Println("  Default loop count: 1")
		fmt.Println("  Progress bar enabled: true")
		fmt.Println("  Verbose logging: false")
		fmt.Println("  WebSocket library: github.com/lxzan/gws")
		fmt.Println("  Metrics library: github.com/hashicorp/go-metrics")
		fmt.Println("  Progress bar: github.com/schollz/progressbar/v3")
	} else {
		fmt.Println("Configuration management - use --show to view current settings")
	}
}

func runHistory(opts *HistoryOptions, globalOpts *GlobalOptions) {
	history, err := loadHistory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading history: %v\n", err)
		os.Exit(1)
	}

	if opts.Clear {
		if err := history.clearHistory(); err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing history: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Test history cleared successfully.")
		return
	}

	// Show history by default if no other action is specified
	if opts.Show || (!opts.Clear) {
		history.printHistory(opts.Limit)
	}
}

func runVisualize(opts *VisualizeOptions, globalOpts *GlobalOptions) {
	history, err := loadHistory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading history: %v\n", err)
		os.Exit(1)
	}

	validMetrics := map[string]bool{
		"success-rate":     true,
		"requests-per-sec": true,
		"avg-latency":      true,
		"throughput":       true,
	}

	if !validMetrics[opts.Metric] {
		fmt.Fprintf(os.Stderr, "Invalid metric: %s. Valid options: success-rate, requests-per-sec, avg-latency, throughput\n", opts.Metric)
		os.Exit(1)
	}

	history.generateComparisonChart(opts.Metric, opts.Limit)
}

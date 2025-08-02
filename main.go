package main

import (
	"fmt"
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

// Commands structure for the CLI
type Commands struct {
	Test   TestOptions   `command:"test" description:"Run a WebSocket load test"`
	Config ConfigOptions `command:"config" description:"Manage configuration"`
}

func main() {
	var globalOpts GlobalOptions
	var commands Commands

	parser := flags.NewParser(&globalOpts, flags.Default)
	parser.AddCommand("test", "Run a WebSocket load test", "Execute a WebSocket load test with specified parameters", &commands.Test)
	parser.AddCommand("config", "Manage configuration", "View and modify tool configuration", &commands.Config)

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
  ws-load config --show`

	// Parse command line arguments
	_, err := parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
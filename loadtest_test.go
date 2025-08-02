package main

import (
	"testing"
	"time"
)

func TestValidateTestOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    *TestOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "10s",
				Connections: 10,
				Message:     "Hello",
				Loop:        1,
			},
			wantErr: false,
		},
		{
			name: "invalid URL",
			opts: &TestOptions{
				URL:         "ws://invalid url with spaces",
				Duration:    "10s",
				Connections: 10,
				Message:     "Hello",
				Loop:        1,
			},
			wantErr: true,
		},
		{
			name: "invalid duration",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "invalid",
				Connections: 10,
				Message:     "Hello",
				Loop:        1,
			},
			wantErr: true,
		},
		{
			name: "zero connections",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "10s",
				Connections: 0,
				Message:     "Hello",
				Loop:        1,
			},
			wantErr: true,
		},
		{
			name: "zero loop count",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "10s",
				Connections: 10,
				Message:     "Hello",
				Loop:        0,
			},
			wantErr: true,
		},
		{
			name: "empty message",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "10s",
				Connections: 10,
				Message:     "",
				Loop:        1,
			},
			wantErr: true,
		},
		{
			name: "valid JSON message",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "10s",
				Connections: 10,
				Message:     `{"type":"ping","data":"test"}`,
				Loop:        1,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON message",
			opts: &TestOptions{
				URL:         "ws://echo.websocket.org",
				Duration:    "10s",
				Connections: 10,
				Message:     `{"type":"ping","data":"test"`,
				Loop:        1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTestOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTestOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWebSocketURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid ws URL",
			url:     "ws://echo.websocket.org",
			want:    "ws://echo.websocket.org",
			wantErr: false,
		},
		{
			name:    "valid wss URL",
			url:     "wss://echo.websocket.org",
			want:    "wss://echo.websocket.org",
			wantErr: false,
		},
		{
			name:    "URL without scheme",
			url:     "echo.websocket.org",
			want:    "ws://echo.websocket.org",
			wantErr: false,
		},
		{
			name:    "invalid URL with special chars",
			url:     "ws://invalid url with spaces",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateWebSocketURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWebSocketURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateWebSocketURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "valid JSON object",
			str:  `{"key":"value"}`,
			want: true,
		},
		{
			name: "valid JSON array",
			str:  `["item1","item2"]`,
			want: true,
		},
		{
			name: "invalid JSON",
			str:  `{"key":"value"`,
			want: false,
		},
		{
			name: "plain string",
			str:  "Hello, World!",
			want: false,
		},
		{
			name: "empty string",
			str:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidJSON(tt.str); got != tt.want {
				t.Errorf("isValidJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "milliseconds",
			d:    500 * time.Millisecond,
			want: "500ms",
		},
		{
			name: "seconds",
			d:    2*time.Second + 500*time.Millisecond,
			want: "2.50s",
		},
		{
			name: "minutes",
			d:    2*time.Minute + 30*time.Second,
			want: "2m30s",
		},
		{
			name: "hours",
			d:    2*time.Hour + 30*time.Minute,
			want: "2h30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 500,
			want:  "500 B",
		},
		{
			name:  "kilobytes",
			bytes: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "megabytes",
			bytes: 1024 * 1024,
			want:  "1.0 MB",
		},
		{
			name:  "gigabytes",
			bytes: 1024 * 1024 * 1024,
			want:  "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBytes(tt.bytes); got != tt.want {
				t.Errorf("formatBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculatePercentile(t *testing.T) {
	latencies := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
	}

	tests := []struct {
		name       string
		percentile int
		want       time.Duration
	}{
		{
			name:       "P50",
			percentile: 50,
			want:       3 * time.Millisecond,
		},
		{
			name:       "P90",
			percentile: 90,
			want:       5 * time.Millisecond,
		},
		{
			name:       "P10",
			percentile: 10,
			want:       1 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculatePercentile(latencies, tt.percentile); got != tt.want {
				t.Errorf("calculatePercentile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		maxLength int
		want      string
	}{
		{
			name:      "short message",
			message:   "Hello",
			maxLength: 10,
			want:      "Hello",
		},
		{
			name:      "long message",
			message:   "This is a very long message that exceeds the maximum length",
			maxLength: 20,
			want:      "This is a very long ...",
		},
		{
			name:      "exact length",
			message:   "Exactly 10 chars",
			maxLength: 10,
			want:      "Exactly 10...",
		},
		{
			name:      "message longer than max",
			message:   "This message is longer than the limit",
			maxLength: 15,
			want:      "This message is...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeMessage(tt.message, tt.maxLength); got != tt.want {
				t.Errorf("sanitizeMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLoadTest(t *testing.T) {
	opts := &TestOptions{
		URL:         "ws://echo.websocket.org",
		Duration:    "10s",
		Connections: 10,
		Message:     "Hello",
		Loop:        1,
	}

	lt := NewLoadTest(opts)

	if lt.opts != opts {
		t.Errorf("NewLoadTest() opts = %v, want %v", lt.opts, opts)
	}

	if lt.results == nil {
		t.Error("NewLoadTest() results should not be nil")
	}

	if lt.metrics == nil {
		t.Error("NewLoadTest() metrics should not be nil")
	}

	if lt.ctx == nil {
		t.Error("NewLoadTest() ctx should not be nil")
	}
}

func TestLoadTestRunWithInvalidDuration(t *testing.T) {
	opts := &TestOptions{
		URL:         "ws://echo.websocket.org",
		Duration:    "invalid",
		Connections: 10,
		Message:     "Hello",
		Loop:        1,
	}

	lt := NewLoadTest(opts)
	err := lt.Run()

	if err == nil {
		t.Error("LoadTest.Run() should return error for invalid duration")
	}
} 
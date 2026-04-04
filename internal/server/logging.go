package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// Metrics tracks request counts and latencies.
type Metrics struct {
	TotalRequests  atomic.Int64
	TotalErrors    atomic.Int64
	RequestsByPath map[string]*atomic.Int64
	StartTime      time.Time
}

var metrics = &Metrics{
	RequestsByPath: make(map[string]*atomic.Int64),
	StartTime:      time.Now(),
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return metrics
}

type logEntry struct {
	Timestamp  string `json:"ts"`
	Level      string `json:"level"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	LatencyMs  float64 `json:"latency_ms"`
	RemoteAddr string `json:"remote_addr"`
	RequestID  string `json:"request_id,omitempty"`
}

type statusWriter struct {
	http.ResponseWriter
	status int
	written bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.written {
		w.status = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.status = 200
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// StructuredLogger is a chi-compatible middleware that outputs JSON logs.
func StructuredLogger(next http.Handler) http.Handler {
	logLevel := os.Getenv("LOG_LEVEL")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(sw, r)

		latency := time.Since(start)

		metrics.TotalRequests.Add(1)
		if sw.status >= 400 {
			metrics.TotalErrors.Add(1)
		}

		// Skip logging health checks unless debug
		if r.URL.Path == "/health" && logLevel != "debug" {
			return
		}

		entry := logEntry{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Level:      "info",
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			Status:     sw.status,
			LatencyMs:  float64(latency.Microseconds()) / 1000.0,
			RemoteAddr: r.RemoteAddr,
			RequestID:  w.Header().Get("x-ms-request-id"),
		}

		if sw.status >= 500 {
			entry.Level = "error"
		} else if sw.status >= 400 {
			entry.Level = "warn"
		}

		out, _ := json.Marshal(entry)
		fmt.Fprintln(os.Stdout, string(out))
	})
}

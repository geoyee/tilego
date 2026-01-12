// Package util provides utility functions for the tile downloader.
package util

import (
	"strings"
	"sync"
)

// ErrorStats tracks and categorizes download errors.
type ErrorStats struct {
	errors map[string]int
	mu     sync.RWMutex
}

// NewErrorStats creates a new ErrorStats instance.
func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		errors: make(map[string]int),
	}
}

// RecordError categorizes and records an error occurrence.
func (es *ErrorStats) RecordError(err error) {
	if err == nil {
		return
	}

	errStr := err.Error()

	es.mu.Lock()
	defer es.mu.Unlock()

	var simplifiedErr string
	switch {
	case strings.Contains(errStr, "context deadline exceeded"):
		simplifiedErr = "Timeout"
	case strings.Contains(errStr, "connection refused"):
		simplifiedErr = "Connection refused"
	case strings.Contains(errStr, "no such host"):
		simplifiedErr = "DNS resolution failed"
	case strings.Contains(errStr, "i/o timeout"):
		simplifiedErr = "I/O timeout"
	case strings.Contains(errStr, "proxyconnect"):
		simplifiedErr = "Proxy connection failed"
	case strings.Contains(errStr, "tls handshake"):
		simplifiedErr = "TLS handshake failed"
	case strings.Contains(errStr, "HTTP 403"):
		simplifiedErr = "HTTP 403 Forbidden"
	case strings.Contains(errStr, "HTTP 404"):
		simplifiedErr = "HTTP 404 Not Found"
	case strings.Contains(errStr, "HTTP 429"):
		simplifiedErr = "HTTP 429 Too Many Requests"
	case strings.Contains(errStr, "HTTP 5"):
		simplifiedErr = "HTTP 5xx Server Error"
	default:
		if len(errStr) > 50 {
			simplifiedErr = errStr[:50] + "..."
		} else {
			simplifiedErr = errStr
		}
	}

	es.errors[simplifiedErr]++
}

// GetErrorStats returns a copy of the current error statistics.
func (es *ErrorStats) GetErrorStats() map[string]int {
	es.mu.RLock()
	defer es.mu.RUnlock()

	stats := make(map[string]int)
	for err, count := range es.errors {
		stats[err] = count
	}

	return stats
}

// HasErrors checks if any errors have been recorded.
func (es *ErrorStats) HasErrors() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return len(es.errors) > 0
}

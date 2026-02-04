package util

import (
	"sync"
)

type ErrorStats struct {
	errors map[string]int
	mu     sync.RWMutex
}

func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		errors: make(map[string]int),
	}
}

func (es *ErrorStats) RecordError(err error) {
	if err == nil {
		return
	}
	es.mu.Lock()
	defer es.mu.Unlock()
	es.errors[err.Error()]++
}

func (es *ErrorStats) GetErrorStats() map[string]int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	stats := make(map[string]int)
	for err, count := range es.errors {
		stats[err] = count
	}

	return stats
}

func (es *ErrorStats) HasErrors() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return len(es.errors) > 0
}

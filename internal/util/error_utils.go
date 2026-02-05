package util

import (
	"sync"
)

type ErrorStats struct {
	Errors map[string]int
	Mu     sync.RWMutex
}

func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		Errors: make(map[string]int),
	}
}

func (es *ErrorStats) RecordError(err error) {
	if err == nil {
		return
	}
	es.Mu.Lock()
	defer es.Mu.Unlock()
	es.Errors[err.Error()]++
}

func (es *ErrorStats) GetErrorStats() map[string]int {
	es.Mu.RLock()
	defer es.Mu.RUnlock()
	stats := make(map[string]int)
	for err, count := range es.Errors {
		stats[err] = count
	}
	return stats
}

func (es *ErrorStats) HasErrors() bool {
	es.Mu.RLock()
	defer es.Mu.RUnlock()
	return len(es.Errors) > 0
}

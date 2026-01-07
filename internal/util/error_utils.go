// Package util 提供工具函数
package util

import (
	"strings"
	"sync"
)

// ErrorStats 错误统计
type ErrorStats struct {
	errors map[string]int
	mu     sync.RWMutex
}

// NewErrorStats 创建错误统计
func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		errors: make(map[string]int),
	}
}

// RecordError 记录错误
func (es *ErrorStats) RecordError(err error) {
	if err == nil {
		return
	}

	errStr := err.Error()

	es.mu.Lock()
	defer es.mu.Unlock()

	// 简化错误信息，提取关键部分
	var simplifiedErr string
	switch {
	case strings.Contains(errStr, "context deadline exceeded"):
		simplifiedErr = "超时"
	case strings.Contains(errStr, "connection refused"):
		simplifiedErr = "连接被拒绝"
	case strings.Contains(errStr, "no such host"):
		simplifiedErr = "域名解析失败"
	case strings.Contains(errStr, "i/o timeout"):
		simplifiedErr = "IO超时"
	case strings.Contains(errStr, "proxyconnect"):
		simplifiedErr = "代理连接失败"
	case strings.Contains(errStr, "tls handshake"):
		simplifiedErr = "TLS握手失败"
	case strings.Contains(errStr, "HTTP 403"):
		simplifiedErr = "HTTP 403禁止访问"
	case strings.Contains(errStr, "HTTP 404"):
		simplifiedErr = "HTTP 404未找到"
	case strings.Contains(errStr, "HTTP 429"):
		simplifiedErr = "HTTP 429请求过多"
	case strings.Contains(errStr, "HTTP 5"):
		simplifiedErr = "HTTP 5xx服务器错误"
	default:
		// 只取错误信息的前50个字符
		if len(errStr) > 50 {
			simplifiedErr = errStr[:50] + "..."
		} else {
			simplifiedErr = errStr
		}
	}

	es.errors[simplifiedErr]++
}

// GetErrorStats 获取错误统计
func (es *ErrorStats) GetErrorStats() map[string]int {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// 返回副本
	stats := make(map[string]int)
	for err, count := range es.errors {
		stats[err] = count
	}

	return stats
}

// HasErrors 检查是否有错误
func (es *ErrorStats) HasErrors() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return len(es.errors) > 0
}

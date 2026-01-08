// Package stats 提供下载统计和监控功能
package stats

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/geoyee/tilego/internal/model"
)

// StatsMonitor 统计监控器
type StatsMonitor struct {
	stats      *model.DownloadStats
	mu         sync.RWMutex
	stopChan   chan bool
	speedMutex sync.RWMutex
}

// NewStatsMonitor 创建统计监控器
func NewStatsMonitor() *StatsMonitor {
	return &StatsMonitor{
		stats:    &model.DownloadStats{},
		stopChan: make(chan bool),
	}
}

// InitStats 初始化统计信息
func (sm *StatsMonitor) InitStats(totalTiles int) {
	sm.stats = &model.DownloadStats{
		Total:        int64(totalTiles),
		StartTime:    time.Now(),
		SpeedHistory: make([]model.SpeedRecord, 0, 100),
	}
}

// GetStats 获取统计信息
func (sm *StatsMonitor) GetStats() *model.DownloadStats {
	return sm.stats
}

// StartMonitoring 开始监控
func (sm *StatsMonitor) StartMonitoring() {
	go sm.monitorStats()
}

// StopMonitoring 停止监控
func (sm *StatsMonitor) StopMonitoring() {
	close(sm.stopChan)
}

// monitorStats 监控统计信息
func (sm *StatsMonitor) monitorStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastSuccess int64
	var lastBytes int64
	lastTime := time.Now()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			duration := now.Sub(lastTime).Seconds()

			currentSuccess := atomic.LoadInt64(&sm.stats.Success)
			currentBytes := atomic.LoadInt64(&sm.stats.BytesTotal)
			currentFailed := atomic.LoadInt64(&sm.stats.Failed)
			currentSkipped := atomic.LoadInt64(&sm.stats.Skipped)

			successDiff := currentSuccess - lastSuccess
			bytesDiff := currentBytes - lastBytes

			var speed float64
			var countSpeed float64
			if duration > 0 {
				speed = float64(bytesDiff) / 1024 / duration
				countSpeed = float64(successDiff) / duration
			}

			sm.speedMutex.Lock()
			sm.stats.SpeedHistory = append(sm.stats.SpeedHistory, model.SpeedRecord{
				Time:  now,
				Speed: speed,
				Count: int64(countSpeed),
			})

			if len(sm.stats.SpeedHistory) > 100 {
				sm.stats.SpeedHistory = sm.stats.SpeedHistory[1:]
			}
			sm.speedMutex.Unlock()

			totalProcessed := currentSuccess + currentSkipped
			percent := float64(totalProcessed) / float64(sm.stats.Total) * 100

			activeWorkers := atomic.LoadInt32(&sm.stats.ActiveWorkers)

			log.Printf("进度: %d/%d (%.1f%%) | 速度: %.1f KB/s, %.1f 瓦片/秒 | 活跃线程: %d | 成功: %d | 失败: %d | 跳过: %d",
				totalProcessed, sm.stats.Total, percent, speed, countSpeed, activeWorkers,
				currentSuccess, currentFailed, currentSkipped)

			lastSuccess = currentSuccess
			lastBytes = currentBytes
			lastTime = now

			if totalProcessed >= sm.stats.Total {
				return
			}
		case <-sm.stopChan:
			return
		}
	}
}

// PrintFinalStats 打印最终统计信息
func (sm *StatsMonitor) PrintFinalStats() {
	duration := time.Since(sm.stats.StartTime)
	totalProcessed := sm.stats.Success + sm.stats.Skipped
	percent := float64(totalProcessed) / float64(sm.stats.Total) * 100

	log.Println("========================================")
	log.Println("下载完成！最终统计")
	log.Println("========================================")
	log.Printf("总耗时: %s", duration.Round(time.Second))
	log.Printf("总瓦片数: %d", sm.stats.Total)
	log.Printf("成功下载: %d", sm.stats.Success)
	log.Printf("跳过已存在: %d", sm.stats.Skipped)
	log.Printf("下载失败: %d", sm.stats.Failed)
	log.Printf("重试次数: %d", sm.stats.Retries)
	log.Printf("下载总大小: %.2f MB", float64(sm.stats.BytesTotal)/1024/1024)
	log.Printf("完成进度: %.1f%%", percent)

	if duration.Seconds() > 0 {
		avgSpeed := float64(sm.stats.BytesTotal) / 1024 / duration.Seconds()
		avgTileSpeed := float64(sm.stats.Success) / duration.Seconds()
		log.Printf("平均速度: %.1f KB/s, %.1f 瓦片/秒", avgSpeed, avgTileSpeed)
	}
	log.Println("========================================")
}

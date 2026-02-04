package stats

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/geoyee/tilego/internal/model"
)

type StatsMonitor struct {
	stats      *model.DownloadStats
	stopChan   chan bool
	speedMutex sync.RWMutex
}

func NewStatsMonitor() *StatsMonitor {
	return &StatsMonitor{
		stats:    &model.DownloadStats{},
		stopChan: make(chan bool),
	}
}

func (sm *StatsMonitor) InitStats(totalTiles int) {
	sm.stats = &model.DownloadStats{
		Total:        int64(totalTiles),
		StartTime:    time.Now(),
		SpeedHistory: make([]model.SpeedRecord, 0, 100),
	}
}

func (sm *StatsMonitor) GetStats() *model.DownloadStats {
	return sm.stats
}

func (sm *StatsMonitor) StartMonitoring() {
	go sm.MonitorStats()
}

func (sm *StatsMonitor) StopMonitoring() {
	close(sm.stopChan)
}

func (sm *StatsMonitor) MonitorStats() {
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

			log.Printf("Progress: %d/%d (%.1f%%) | Speed: %.1f KB/s, %.1f tiles/sec | Active threads: %d | Success: %d | Failed: %d | Skipped: %d",
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

func (sm *StatsMonitor) PrintFinalStats() {
	duration := time.Since(sm.stats.StartTime)
	totalProcessed := sm.stats.Success + sm.stats.Skipped
	percent := float64(totalProcessed) / float64(sm.stats.Total) * 100

	log.Println("========================================")
	log.Println("Download Complete! Final Statistics")
	log.Println("========================================")
	log.Printf("Total time: %s", duration.Round(time.Second))
	log.Printf("Total tiles: %d", sm.stats.Total)
	log.Printf("Successfully downloaded: %d", sm.stats.Success)
	log.Printf("Skipped (existing): %d", sm.stats.Skipped)
	log.Printf("Failed: %d", sm.stats.Failed)
	log.Printf("Retry count: %d", sm.stats.Retries)
	log.Printf("Total downloaded: %.2f MB", float64(sm.stats.BytesTotal)/1024/1024)
	log.Printf("Completion: %.1f%%", percent)

	if duration.Seconds() > 0 {
		avgSpeed := float64(sm.stats.BytesTotal) / 1024 / duration.Seconds()
		avgTileSpeed := float64(sm.stats.Success) / duration.Seconds()
		log.Printf("Average speed: %.1f KB/s, %.1f tiles/sec", avgSpeed, avgTileSpeed)
	}
	log.Println("========================================")
}

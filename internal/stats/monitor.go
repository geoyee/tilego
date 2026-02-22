package stats

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/geoyee/tilego/internal/model"
)

type StatsMonitor struct {
	Stats      *model.DownloadStats
	StopChan   chan bool
	SpeedMutex sync.RWMutex
}

func NewStatsMonitor() *StatsMonitor {
	return &StatsMonitor{
		Stats:    &model.DownloadStats{},
		StopChan: make(chan bool),
	}
}

func (sm *StatsMonitor) InitStats(totalTiles int) {
	sm.Stats = &model.DownloadStats{
		Total:        int64(totalTiles),
		StartTime:    time.Now(),
		SpeedHistory: make([]model.SpeedRecord, 0, 100),
	}
}

func (sm *StatsMonitor) GetStats() *model.DownloadStats {
	return sm.Stats
}

func (sm *StatsMonitor) StartMonitoring() {
	go sm.MonitorStats()
}

func (sm *StatsMonitor) StopMonitoring() {
	close(sm.StopChan)
}

func (sm *StatsMonitor) MonitorStats() {
	ticker := time.NewTicker(10 * time.Second) // 10秒一次的定时器
	defer ticker.Stop()
	var lastSuccess int64
	var lastBytes int64
	lastTime := time.Now()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			duration := now.Sub(lastTime).Seconds()
			currentSuccess := atomic.LoadInt64(&sm.Stats.Success)
			currentBytes := atomic.LoadInt64(&sm.Stats.BytesTotal)
			currentFailed := atomic.LoadInt64(&sm.Stats.Failed)
			currentSkipped := atomic.LoadInt64(&sm.Stats.Skipped)
			successDiff := currentSuccess - lastSuccess
			bytesDiff := currentBytes - lastBytes
			var speed float64
			var countSpeed float64
			if duration > 0 {
				speed = float64(bytesDiff) / 1024 / duration
				countSpeed = float64(successDiff) / duration
			}
			sm.SpeedMutex.Lock()
			sm.Stats.SpeedHistory = append(sm.Stats.SpeedHistory, model.SpeedRecord{
				Time:  now,
				Speed: speed,
				Count: int64(countSpeed),
			})
			if len(sm.Stats.SpeedHistory) > 100 {
				sm.Stats.SpeedHistory = sm.Stats.SpeedHistory[1:]
			}
			sm.SpeedMutex.Unlock()
			totalProcessed := currentSuccess + currentSkipped
			percent := float64(totalProcessed) / float64(sm.Stats.Total) * 100
			activeWorkers := atomic.LoadInt32(&sm.Stats.ActiveWorkers)
			log.Printf("Progress: %d/%d (%.1f%%) | Speed: %.1f KB/s, %.1f tiles/sec | Active threads: %d | Success: %d | Failed: %d | Skipped: %d",
				totalProcessed, sm.Stats.Total, percent, speed, countSpeed, activeWorkers,
				currentSuccess, currentFailed, currentSkipped)
			lastSuccess = currentSuccess
			lastBytes = currentBytes
			lastTime = now
			if totalProcessed >= sm.Stats.Total {
				return
			}
		case <-sm.StopChan:
			return
		}
	}
}

func (sm *StatsMonitor) PrintFinalStats() {
	duration := time.Since(sm.Stats.StartTime)
	totalProcessed := sm.Stats.Success + sm.Stats.Skipped + sm.Stats.Failed
	percent := float64(totalProcessed) / float64(sm.Stats.Total) * 100
	log.Println("========================================")
	if totalProcessed >= sm.Stats.Total {
		log.Println("Download Complete! Final Statistics")
	} else {
		log.Println("Download Incomplete! Final Statistics")
	}
	log.Println("========================================")
	log.Printf("Total time: %s", duration.Round(time.Second))
	log.Printf("Total tiles: %d", sm.Stats.Total)
	log.Printf("Successfully downloaded: %d", sm.Stats.Success)
	log.Printf("Skipped (existing): %d", sm.Stats.Skipped)
	log.Printf("Failed: %d", sm.Stats.Failed)
	log.Printf("Not processed: %d", sm.Stats.Total-totalProcessed)
	log.Printf("Retry count: %d", sm.Stats.Retries)
	log.Printf("Total downloaded: %.2f MB", float64(sm.Stats.BytesTotal)/1024/1024)
	log.Printf("Completion: %.1f%%", percent)
	if duration.Seconds() > 0 {
		avgSpeed := float64(sm.Stats.BytesTotal) / 1024 / duration.Seconds()
		avgTileSpeed := float64(sm.Stats.Success) / duration.Seconds()
		log.Printf("Average speed: %.1f KB/s, %.1f tiles/sec", avgSpeed, avgTileSpeed)
	}
	log.Println("========================================")
}

// Package download 提供下载相关功能
package download

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/geoyee/tilego/internal/model"
)

// WorkerPool 工作池
type WorkerPool struct {
	workers    int
	taskQueue  chan *model.DownloadTask
	wg         sync.WaitGroup
	downloader *Downloader
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, downloader *Downloader) *WorkerPool {
	return &WorkerPool{
		workers:    workers,
		taskQueue:  make(chan *model.DownloadTask, workers*2),
		wg:         sync.WaitGroup{},
		downloader: downloader,
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start(stats *model.DownloadStats) {
	wp.wg.Add(wp.workers)
	for i := 0; i < wp.workers; i++ {
		go func() {
			defer wp.wg.Done()
			for task := range wp.taskQueue {
				atomic.AddInt32(&stats.ActiveWorkers, 1)
				wp.downloader.DownloadTask(task, stats)
				atomic.AddInt32(&stats.ActiveWorkers, -1)
			}
		}()
	}
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	close(wp.taskQueue)
	wp.wg.Wait()
}

// SubmitTask 提交任务
func (wp *WorkerPool) SubmitTask(task *model.DownloadTask) {
	wp.taskQueue <- task
}

// SubmitTasksInBatches 批量提交任务
func (wp *WorkerPool) SubmitTasksInBatches(tiles []model.Tile, batchSize int, downloader *Downloader, stats *model.DownloadStats) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	batchCount := 0
	totalTiles := len(tiles)
	submitted := 0

	for i := 0; i < totalTiles; i += batchSize {
		end := i + batchSize
		if end > totalTiles {
			end = totalTiles
		}

		batch := tiles[i:end]
		for _, tile := range batch {
			url := downloader.GetTileURL(tile)
			savePath, err := downloader.GetSavePath(tile)
			if err != nil {
				log.Printf("生成保存路径失败: %v", err)
				atomic.AddInt64(&stats.Failed, 1)
				continue
			}

			task := &model.DownloadTask{
				Tile:     tile,
				URL:      url,
				SavePath: savePath,
				Retry:    0,
				Priority: 0,
			}

			wp.SubmitTask(task)
			submitted++
		}

		batchCount++
		if batchCount%10 == 0 {
			log.Printf("已提交 %d/%d 瓦片任务", submitted, totalTiles)
		}
	}

	log.Printf("所有任务提交完成，总计 %d 个瓦片", submitted)
}

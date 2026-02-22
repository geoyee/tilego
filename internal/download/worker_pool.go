package download

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/geoyee/tilego/internal/model"
)

type WorkerPool struct {
	Workers    int
	TaskQueue  chan *model.DownloadTask
	Wg         sync.WaitGroup
	Downloader *Downloader
	stopped    atomic.Bool
	stopOnce   sync.Once
	stopChan   chan struct{}
}

func NewWorkerPool(workers int, downloader *Downloader) *WorkerPool {
	return &WorkerPool{
		Workers:    workers,
		TaskQueue:  make(chan *model.DownloadTask, workers*2),
		Wg:         sync.WaitGroup{},
		Downloader: downloader,
		stopChan:   make(chan struct{}),
	}
}

func (wp *WorkerPool) Start(stats *model.DownloadStats) {
	wp.Wg.Add(wp.Workers)
	for i := 0; i < wp.Workers; i++ {
		go func() {
			defer wp.Wg.Done()
			for task := range wp.TaskQueue {
				atomic.AddInt32(&stats.ActiveWorkers, 1)
				wp.Downloader.DownloadTask(task, stats)
				atomic.AddInt32(&stats.ActiveWorkers, -1)
			}
		}()
	}
}

func (wp *WorkerPool) Stop() {
	wp.stopOnce.Do(func() {
		wp.stopped.Store(true)
		close(wp.stopChan)
		close(wp.TaskQueue)
	})
	wp.Wg.Wait()
}

func (wp *WorkerPool) SubmitTask(task *model.DownloadTask) bool {
	if wp.stopped.Load() {
		return false
	}
	select {
	case <-wp.stopChan:
		return false
	case wp.TaskQueue <- task:
		return true
	}
}

func (wp *WorkerPool) SubmitTasksInBatches(tiles []model.Tile, batchSize int, downloader *Downloader, stats *model.DownloadStats) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	batchCount := 0
	totalTiles := len(tiles)
	submitted := 0
	for i := 0; i < totalTiles; i += batchSize {
		if wp.stopped.Load() || downloader.stopped.Load() {
			log.Printf("Task submission stopped at %d/%d tiles", submitted, totalTiles)
			return
		}
		end := min(i+batchSize, totalTiles)
		batch := tiles[i:end]
		for _, tile := range batch {
			if wp.stopped.Load() || downloader.stopped.Load() {
				log.Printf("Task submission stopped at %d/%d tiles", submitted, totalTiles)
				return
			}
			url := downloader.GetTileURL(tile)
			savePath, err := downloader.GetSavePath(tile)
			if err != nil {
				log.Printf("Failed to generate save path: %v", err)
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
			if !wp.SubmitTask(task) {
				log.Printf("Task submission stopped at %d/%d tiles", submitted, totalTiles)
				return
			}
			submitted++
		}
		batchCount++
		if batchCount%10 == 0 {
			log.Printf("Submitted %d/%d tile tasks", submitted, totalTiles)
		}
	}
	log.Printf("All tasks submitted. Total tiles: %d", submitted)
}

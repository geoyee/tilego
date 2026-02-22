package download

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geoyee/tilego/internal/model"
)

func TestWorkerPool_SubmitTask(t *testing.T) {
	downloader := &Downloader{
		Config: &model.Config{},
	}
	wp := NewWorkerPool(2, downloader)

	var processed int64

	wp.Wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wp.Wg.Done()
			for range wp.TaskQueue {
				atomic.AddInt64(&processed, 1)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		task := &model.DownloadTask{
			Tile: model.Tile{X: i, Y: i, Z: 1},
			URL:  "http://example.com/test",
		}
		if !wp.SubmitTask(task) {
			t.Errorf("SubmitTask returned false for task %d", i)
		}
	}

	wp.Stop()

	if processed != 10 {
		t.Errorf("Expected 10 processed tasks, got %d", processed)
	}
}

func TestWorkerPool_StopDuringSubmit(t *testing.T) {
	downloader := &Downloader{
		Config: &model.Config{},
	}
	wp := NewWorkerPool(1, downloader)

	var started atomic.Bool
	var wg sync.WaitGroup

	wp.Wg.Add(1)
	go func() {
		defer wp.Wg.Done()
		started.Store(true)
		for range wp.TaskQueue {
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for !started.Load() {
		time.Sleep(1 * time.Millisecond)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		wp.Stop()
	}()

	time.Sleep(5 * time.Millisecond)

	task := &model.DownloadTask{
		Tile: model.Tile{X: 1, Y: 1, Z: 1},
		URL:  "http://example.com/test",
	}
	wp.SubmitTask(task)

	wg.Wait()

	if !wp.stopped.Load() {
		t.Error("WorkerPool should be stopped")
	}
}

func TestWorkerPool_SubmitAfterStop(t *testing.T) {
	downloader := &Downloader{
		Config: &model.Config{},
	}
	wp := NewWorkerPool(1, downloader)

	wp.Wg.Add(1)
	go func() {
		defer wp.Wg.Done()
		for range wp.TaskQueue {
		}
	}()

	wp.Stop()

	task := &model.DownloadTask{
		Tile: model.Tile{X: 1, Y: 1, Z: 1},
		URL:  "http://example.com/test",
	}

	if wp.SubmitTask(task) {
		t.Error("SubmitTask should return false after Stop")
	}
}

func TestWorkerPool_MultipleStop(t *testing.T) {
	downloader := &Downloader{
		Config: &model.Config{},
	}
	wp := NewWorkerPool(1, downloader)

	wp.Wg.Add(1)
	go func() {
		defer wp.Wg.Done()
		for range wp.TaskQueue {
		}
	}()

	for i := 0; i < 10; i++ {
		wp.Stop()
	}

	if !wp.stopped.Load() {
		t.Error("WorkerPool should be stopped")
	}
}

func TestWorkerPool_ConcurrentSubmitAndStop(t *testing.T) {
	downloader := &Downloader{
		Config: &model.Config{},
	}
	wp := NewWorkerPool(5, downloader)

	var submittedCount int64
	var wg sync.WaitGroup

	wp.Wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wp.Wg.Done()
			for range wp.TaskQueue {
			}
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				task := &model.DownloadTask{
					Tile: model.Tile{X: id, Y: j, Z: 1},
					URL:  "http://example.com/test",
				}
				if wp.SubmitTask(task) {
					atomic.AddInt64(&submittedCount, 1)
				}
			}
		}(i)
	}

	time.Sleep(10 * time.Millisecond)
	wp.Stop()
	wg.Wait()

	t.Logf("Submitted %d tasks before stop", submittedCount)
}

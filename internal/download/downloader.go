package download

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/geoyee/tilego/internal/calculator"
	"github.com/geoyee/tilego/internal/client"
	"github.com/geoyee/tilego/internal/model"
	"github.com/geoyee/tilego/internal/resume"
	"github.com/geoyee/tilego/internal/stats"
	"github.com/geoyee/tilego/internal/util"
)

type Downloader struct {
	Config        *model.Config
	HttpClient    *client.HTTPClient
	Calculator    *calculator.TileCalculator
	ResumeManager *resume.ResumeManager
	StatsMonitor  *stats.StatsMonitor
	ErrorStats    *util.ErrorStats
	WorkerPool    *WorkerPool
	Limiter       *rate.Limiter
	FileExtension string
	stopped       atomic.Bool
}

func NewDownloader(config *model.Config) *Downloader {
	return &Downloader{
		Config:     config,
		ErrorStats: util.NewErrorStats(),
	}
}

func (d *Downloader) Init() error {
	httpConfig := &client.Config{
		Timeout:   d.Config.Timeout,
		ProxyURL:  d.Config.ProxyURL,
		UseHTTP2:  d.Config.UseHTTP2,
		KeepAlive: d.Config.KeepAlive,
		UserAgent: d.Config.UserAgent,
	}
	d.HttpClient = client.NewHTTPClient(httpConfig)
	if d.Config.ProxyURL != "" {
		if err := d.HttpClient.TestProxyConnection(); err != nil {
			log.Printf("Warning: Proxy connection test failed: %v", err)
		} else {
			log.Println("Proxy connection test successful")
		}
	}
	if d.Config.RateLimit > 0 {
		d.Limiter = rate.NewLimiter(rate.Limit(d.Config.RateLimit), d.Config.RateLimit)
		log.Printf("Rate limit: %d requests/second", d.Config.RateLimit)
	}
	d.Calculator = calculator.NewTileCalculator()
	d.ResumeManager = resume.NewResumeManager(d.Config.SaveDir, d.Config.ResumeFile)
	if err := d.ResumeManager.LoadResumeData(); err != nil {
		return fmt.Errorf("failed to load resume data: %w", err)
	}
	d.StatsMonitor = stats.NewStatsMonitor()
	d.WorkerPool = NewWorkerPool(d.Config.Threads, d)
	log.Printf("Concurrent threads: %d", d.Config.Threads)
	d.FileExtension = util.GetFileExtension(d.Config.URLTemplate, "")
	return nil
}

func (d *Downloader) Cleanup() {
	if d.HttpClient != nil {
		if transport, ok := d.HttpClient.GetClient().Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	if d.StatsMonitor != nil && d.StatsMonitor.GetStats().Total > 0 {
		if err := d.ResumeManager.SaveResumeData(d.Config.URLTemplate, d.Config.Format, int(d.StatsMonitor.GetStats().Total)); err != nil {
			log.Printf("Failed to save resume data during cleanup: %v", err)
		}
	}
	d.printErrorStats()
}

func (d *Downloader) Stop() {
	d.stopped.Store(true)
	if d.WorkerPool != nil {
		d.WorkerPool.Stop()
	}
}

func (d *Downloader) Run() error {
	defer d.Cleanup()
	if err := d.Init(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	if err := d.Calculator.ValidateZoomRange(d.Config.MinZoom, d.Config.MaxZoom); err != nil {
		return err
	}
	if err := d.Calculator.ValidateLatLonRange(d.Config.MinLon, d.Config.MinLat, d.Config.MaxLon, d.Config.MaxLat); err != nil {
		return err
	}
	log.Println("Calculating tile range...")
	tiles := d.Calculator.CalculateTiles(d.Config.MinLon, d.Config.MinLat, d.Config.MaxLon, d.Config.MaxLat, d.Config.MinZoom, d.Config.MaxZoom)
	if len(tiles) == 0 {
		return calculator.ErrNoTilesFound
	}
	if err := util.EnsureDirExists(d.Config.SaveDir); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}
	d.StatsMonitor.InitStats(len(tiles))
	log.Printf("Starting download of %d tiles", len(tiles))
	log.Printf("Save directory: %s", d.Config.SaveDir)
	log.Printf("Save format: %s", d.Config.Format)
	log.Printf("HTTP/2: %v, Keep-Alive: %v", d.Config.UseHTTP2, d.Config.KeepAlive)
	if d.Config.ProxyURL != "" {
		log.Printf("Using proxy: %s", d.Config.ProxyURL)
	}
	if d.Config.RateLimit > 0 {
		log.Printf("Rate limit: %d requests/second", d.Config.RateLimit)
	}
	d.StatsMonitor.StartMonitoring()
	d.WorkerPool.Start(d.StatsMonitor.GetStats())
	d.WorkerPool.SubmitTasksInBatches(tiles, d.Config.BatchSize, d, d.StatsMonitor.GetStats())
	d.WorkerPool.Stop()
	d.StatsMonitor.StopMonitoring()
	time.Sleep(500 * time.Millisecond)
	d.StatsMonitor.PrintFinalStats()
	if d.StatsMonitor.GetStats().Failed > 0 {
		return fmt.Errorf("%d tiles failed to download", d.StatsMonitor.GetStats().Failed)
	}
	return nil
}

func (d *Downloader) GetTileURL(tile model.Tile) string {
	return util.GetTileURL(d.Config.URLTemplate, tile.X, tile.Y, tile.Z)
}

func (d *Downloader) GetSavePath(tile model.Tile) (string, error) {
	return util.GetSavePath(d.Config.SaveDir, d.Config.Format, tile.X, tile.Y, tile.Z, d.FileExtension)
}

func (d *Downloader) DownloadTask(task *model.DownloadTask, stats *model.DownloadStats) {
	if d.stopped.Load() {
		return
	}
	if d.Limiter != nil {
		if err := d.Limiter.Wait(context.Background()); err != nil {
			log.Printf("Rate limit error: %v", err)
			return
		}
	}
	if d.stopped.Load() {
		return
	}
	if d.Config.SkipExisting {
		if downloaded, info := d.ResumeManager.IsTileDownloaded(task.Tile); downloaded {
			atomic.AddInt64(&stats.Skipped, 1)
			if info != nil {
				atomic.AddInt64(&stats.BytesTotal, info.FileSize)
			}
			return
		}
	}
	var lastErr error
	maxAttempts := d.Config.Retries + 1
	for attempt := range maxAttempts {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
			delay = min(delay, 30*time.Second)
			time.Sleep(delay)
			atomic.AddInt64(&stats.Retries, 1)
		}
		err := d.doDownload(task)
		if err == nil {
			atomic.AddInt64(&stats.Success, 1)
			if stat, err := os.Stat(task.SavePath); err == nil {
				atomic.AddInt64(&stats.BytesTotal, stat.Size())
			}
			return
		}
		lastErr = err
		d.ErrorStats.RecordError(err)
		errStr := err.Error()
		if isClientError(errStr) {
			break
		}

		if isNetworkError(errStr) {
			continue
		}
	}
	atomic.AddInt64(&stats.Failed, 1)
	d.ResumeManager.MarkTileFailed(task.Tile, lastErr.Error())
	if atomic.LoadInt64(&stats.Failed)%10 == 0 {
		log.Printf("Total failed: %d", atomic.LoadInt64(&stats.Failed))
	}
}

func isClientError(errStr string) bool {
	return strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "400")
}

func isNetworkError(errStr string) bool {
	lowerErr := strings.ToLower(errStr)
	return strings.Contains(lowerErr, "timeout") ||
		strings.Contains(lowerErr, "deadline") ||
		strings.Contains(lowerErr, "connection") ||
		strings.Contains(lowerErr, "network")
}

func (d *Downloader) doDownload(task *model.DownloadTask) error {
	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", d.Config.UserAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	if d.Config.Referer != "" {
		req.Header.Set("Referer", d.Config.Referer)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := d.HttpClient.GetClient().Do(req)
	if err != nil {
		return err
	}
	defer client.SafeCloseResponse(resp)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	buf := make([]byte, d.Config.BufferSize)
	var data []byte
	var totalRead int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
			totalRead += int64(n)
			if totalRead > d.Config.MaxFileSize {
				return fmt.Errorf("file size exceeds limit: %d > %d", totalRead, d.Config.MaxFileSize)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("failed to read response: %w", readErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	if int64(len(data)) < d.Config.MinFileSize {
		return fmt.Errorf("file too small: %d < %d", len(data), d.Config.MinFileSize)
	}
	if !util.ValidateFileFormat(data, d.Config.MinFileSize, d.Config.MaxFileSize) {
		return fmt.Errorf("invalid file format")
	}
	if err := util.EnsureDirExists(filepath.Dir(task.SavePath)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(task.SavePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}
	var md5Hash string
	if d.Config.CheckMD5 {
		hash := md5.Sum(data)
		md5Hash = hex.EncodeToString(hash[:])
	}
	d.ResumeManager.MarkTileComplete(task.Tile, task.SavePath, int64(len(data)), md5Hash)
	return nil
}

func (d *Downloader) printErrorStats() {
	if !d.ErrorStats.HasErrors() {
		return
	}
	log.Println("Error statistics:")
	for err, count := range d.ErrorStats.GetErrorStats() {
		log.Printf("  %s: %d occurrences", err, count)
	}
}

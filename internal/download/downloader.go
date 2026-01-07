// Package download 提供下载相关功能
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

// Downloader 瓦片下载器
type Downloader struct {
	config        *model.Config
	httpClient    *client.HTTPClient
	calculator    *calculator.TileCalculator
	resumeManager *resume.ResumeManager
	statsMonitor  *stats.StatsMonitor
	errorStats    *util.ErrorStats
	workerPool    *WorkerPool
	limiter       *rate.Limiter
	fileExtension string
}

// NewDownloader 创建下载器实例
func NewDownloader(config *model.Config) *Downloader {
	return &Downloader{
		config:     config,
		errorStats: util.NewErrorStats(),
	}
}

// Init 初始化下载器
func (d *Downloader) Init() error {
	// 初始化HTTP客户端
	httpConfig := &client.Config{
		Timeout:   d.config.Timeout,
		ProxyURL:  d.config.ProxyURL,
		UseHTTP2:  d.config.UseHTTP2,
		KeepAlive: d.config.KeepAlive,
		UserAgent: d.config.UserAgent,
	}
	d.httpClient = client.NewHTTPClient(httpConfig)

	// 测试代理连接
	if d.config.ProxyURL != "" {
		if err := d.httpClient.TestProxyConnection(); err != nil {
			log.Printf("警告: 代理连接测试失败: %v", err)
		} else {
			log.Printf("代理连接测试成功")
		}
	}

	// 初始化速率限制器
	if d.config.RateLimit > 0 {
		d.limiter = rate.NewLimiter(rate.Limit(d.config.RateLimit), d.config.RateLimit)
		log.Printf("速率限制: %d 请求/秒", d.config.RateLimit)
	}

	// 初始化计算器
	d.calculator = calculator.NewTileCalculator()

	// 初始化断点续传管理器
	d.resumeManager = resume.NewResumeManager(d.config.SaveDir, d.config.ResumeFile)
	if err := d.resumeManager.LoadResumeData(); err != nil {
		return fmt.Errorf("加载断点数据失败: %v", err)
	}

	// 初始化统计监控器
	d.statsMonitor = stats.NewStatsMonitor()

	// 初始化工作池
	d.workerPool = NewWorkerPool(d.config.Threads, d)
	log.Printf("并发线程数: %d", d.config.Threads)

	// 计算文件扩展名
	d.fileExtension = util.GetFileExtension(d.config.URLTemplate, d.config.OutputType)

	return nil
}

// Cleanup 清理资源
func (d *Downloader) Cleanup() {
	if d.httpClient != nil {
		if transport, ok := d.httpClient.GetClient().Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}

	if d.statsMonitor != nil && d.statsMonitor.GetStats().Total > 0 {
		if err := d.resumeManager.SaveResumeData(d.config.URLTemplate, d.config.Format, int(d.statsMonitor.GetStats().Total)); err != nil {
			log.Printf("清理时保存断点数据失败: %v", err)
		}
	}

	d.printErrorStats()
}

// Run 运行下载任务
func (d *Downloader) Run() error {
	defer d.Cleanup()

	if err := d.Init(); err != nil {
		return fmt.Errorf("初始化失败: %v", err)
	}

	// 验证参数
	if err := d.calculator.ValidateZoomRange(d.config.MinZoom, d.config.MaxZoom); err != nil {
		return err
	}

	if err := d.calculator.ValidateLatLonRange(d.config.MinLon, d.config.MinLat, d.config.MaxLon, d.config.MaxLat); err != nil {
		return err
	}

	log.Printf("正在并行计算瓦片范围...")
	tiles, err := d.calculator.CalculateTiles(d.config.MinLon, d.config.MinLat, d.config.MaxLon, d.config.MaxLat, d.config.MinZoom, d.config.MaxZoom)
	if err != nil {
		return fmt.Errorf("计算瓦片失败: %v", err)
	}

	if len(tiles) == 0 {
		return calculator.ErrNoTilesFound
	}

	if err := util.EnsureDirExists(d.config.SaveDir); err != nil {
		return fmt.Errorf("创建保存目录失败: %v", err)
	}

	d.statsMonitor.InitStats(len(tiles))

	log.Printf("开始下载 %d 个瓦片", len(tiles))
	log.Printf("保存目录: %s", d.config.SaveDir)
	log.Printf("保存格式: %s", d.config.Format)
	log.Printf("HTTP/2: %v, Keep-Alive: %v", d.config.UseHTTP2, d.config.KeepAlive)
	if d.config.ProxyURL != "" {
		log.Printf("使用代理: %s", d.config.ProxyURL)
	}
	if d.config.RateLimit > 0 {
		log.Printf("速率限制: %d 请求/秒", d.config.RateLimit)
	}

	d.statsMonitor.StartMonitoring()

	d.workerPool.Start(d.statsMonitor.GetStats())

	d.workerPool.SubmitTasksInBatches(tiles, d.config.BatchSize, d, d.statsMonitor.GetStats())

	d.workerPool.Stop()

	d.statsMonitor.StopMonitoring()

	time.Sleep(500 * time.Millisecond)

	d.statsMonitor.PrintFinalStats()

	if d.statsMonitor.GetStats().Failed > 0 {
		return fmt.Errorf("有 %d 个瓦片下载失败", d.statsMonitor.GetStats().Failed)
	}

	return nil
}

// GetTileURL 获取瓦片URL
func (d *Downloader) GetTileURL(tile model.Tile) string {
	return util.GetTileURL(d.config.URLTemplate, tile.X, tile.Y, tile.Z)
}

// GetSavePath 获取保存路径
func (d *Downloader) GetSavePath(tile model.Tile) (string, error) {
	return util.GetSavePath(d.config.SaveDir, d.config.Format, tile.X, tile.Y, tile.Z, d.fileExtension)
}

// DownloadTask 下载单个任务
func (d *Downloader) DownloadTask(task *model.DownloadTask, stats *model.DownloadStats) {
	if d.limiter != nil {
		if err := d.limiter.Wait(context.Background()); err != nil {
			log.Printf("速率限制错误: %v", err)
			return
		}
	}

	if d.config.SkipExisting {
		if downloaded, info := d.resumeManager.IsTileDownloaded(task.Tile); downloaded {
			atomic.AddInt64(&stats.Skipped, 1)
			if info != nil {
				atomic.AddInt64(&stats.BytesTotal, info.FileSize)
			}
			return
		}
	}

	var lastErr error
	maxAttempts := d.config.Retries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
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
		d.errorStats.RecordError(err)

		errStr := err.Error()
		if strings.Contains(errStr, "404") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "400") {
			break
		}

		if strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "deadline") ||
			strings.Contains(errStr, "connection") ||
			strings.Contains(errStr, "network") {
			continue
		}
	}

	atomic.AddInt64(&stats.Failed, 1)
	d.resumeManager.MarkTileFailed(task.Tile, lastErr.Error())

	if atomic.LoadInt64(&stats.Failed)%10 == 0 {
		log.Printf("累计失败数: %d", atomic.LoadInt64(&stats.Failed))
	}
}

// doDownload 执行下载
func (d *Downloader) doDownload(task *model.DownloadTask) error {
	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", d.config.UserAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://example.com/")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.config.Timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := d.httpClient.GetClient().Do(req)
	if err != nil {
		return err
	}
	defer client.SafeCloseResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, d.config.BufferSize)
	var data []byte
	var totalRead int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
			totalRead += int64(n)

			if totalRead > d.config.MaxFileSize {
				return fmt.Errorf("文件大小超过限制: %d > %d", totalRead, d.config.MaxFileSize)
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("读取响应失败: %v", readErr)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if int64(len(data)) < d.config.MinFileSize {
		return fmt.Errorf("文件太小: %d < %d", len(data), d.config.MinFileSize)
	}

	if !util.ValidateFileFormat(data, d.config.MinFileSize, d.config.MaxFileSize) {
		return fmt.Errorf("无效的文件格式")
	}

	if err := util.EnsureDirExists(filepath.Dir(task.SavePath)); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	if err := os.WriteFile(task.SavePath, data, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	var md5Hash string
	if d.config.CheckMD5 {
		hash := md5.Sum(data)
		md5Hash = hex.EncodeToString(hash[:])
	}

	d.resumeManager.MarkTileComplete(task.Tile, task.SavePath, int64(len(data)), md5Hash)

	return nil
}

// printErrorStats 打印错误统计
func (d *Downloader) printErrorStats() {
	if !d.errorStats.HasErrors() {
		return
	}

	log.Println("错误统计:")
	for err, count := range d.errorStats.GetErrorStats() {
		log.Printf("  %s: %d次", err, count)
	}
}

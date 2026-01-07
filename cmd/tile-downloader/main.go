package main

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/time/rate"
)

// 性能优化配置
const (
	MaxIdleConns        = 200
	MaxIdleConnsPerHost = 50
	MaxConnsPerHost     = 50
	IdleConnTimeout     = 30 * time.Second
)

// TileDownloader 瓦片下载器
type TileDownloader struct {
	URLTemplate  string
	MinLon       float64
	MinLat       float64
	MaxLon       float64
	MaxLat       float64
	MinZoom      int
	MaxZoom      int
	SaveDir      string
	Format       string
	Threads      int
	Timeout      int
	Retries      int
	UserAgent    string
	OutputType   string
	ProxyURL     string
	ResumeFile   string
	SkipExisting bool
	CheckMD5     bool
	MinFileSize  int64
	MaxFileSize  int64
	RateLimit    int
	BatchSize    int
	UseHTTP2     bool
	KeepAlive    bool
	BufferSize   int
	ResumeData   *ResumeData
	ResumeMutex  sync.Mutex
	client       *http.Client
	limiter      *rate.Limiter
	workerPool   *WorkerPool
	stats        *DownloadStats
	errorStats   map[string]int
	errorStatsMu sync.RWMutex
}

// DownloadTask 下载任务
type DownloadTask struct {
	Tile     Tile
	URL      string
	SavePath string
	Retry    int
	Priority int
}

// WorkerPool 工作池
type WorkerPool struct {
	workers    int
	taskQueue  chan *DownloadTask
	wg         sync.WaitGroup
	downloader *TileDownloader
}

// Tile 瓦片坐标
type Tile struct {
	X, Y, Z int
}

// DownloadStats 下载统计
type DownloadStats struct {
	Total         int64
	Success       int64
	Failed        int64
	Skipped       int64
	Retries       int64
	BytesTotal    int64
	ActiveWorkers int32
	StartTime     time.Time
	SpeedHistory  []SpeedRecord
	mu            sync.RWMutex
}

// SpeedRecord 速度记录
type SpeedRecord struct {
	Time  time.Time
	Speed float64
	Count int64
}

// ResumeData 断点续传数据
type ResumeData struct {
	Version      string              `json:"version"`
	URLTemplate  string              `json:"url_template"`
	SaveDir      string              `json:"save_dir"`
	Format       string              `json:"format"`
	Completed    map[string]TileInfo `json:"completed"`
	Failed       map[string]string   `json:"failed"`
	TotalTiles   int                 `json:"total_tiles"`
	DownloadTime time.Time           `json:"download_time"`
}

// TileInfo 瓦片信息
type TileInfo struct {
	X          int       `json:"x"`
	Y          int       `json:"y"`
	Z          int       `json:"z"`
	FilePath   string    `json:"file_path"`
	FileSize   int64     `json:"file_size"`
	MD5        string    `json:"md5,omitempty"`
	Downloaded time.Time `json:"downloaded"`
	Valid      bool      `json:"valid"`
}

// NewTileDownloader 创建下载器实例
func NewTileDownloader() *TileDownloader {
	return &TileDownloader{
		Threads:      20, // 降低默认线程数
		Timeout:      60, // 增加超时时间
		Retries:      5,  // 增加重试次数
		UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Format:       "zxy",
		OutputType:   "auto",
		ResumeFile:   ".resume.json",
		SkipExisting: true,
		MinFileSize:  100,
		MaxFileSize:  2 * 1024 * 1024,
		BatchSize:    200, // 降低批量大小
		UseHTTP2:     true,
		KeepAlive:    true,
		BufferSize:   64 * 1024,
		RateLimit:    10, // 降低默认速率限制
		errorStats:   make(map[string]int),
	}
}

// createHTTPClient 创建HTTP客户端
func (d *TileDownloader) createHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     d.UseHTTP2,
		MaxIdleConns:          MaxIdleConns,
		MaxIdleConnsPerHost:   MaxIdleConnsPerHost,
		MaxConnsPerHost:       MaxConnsPerHost,
		IdleConnTimeout:       IdleConnTimeout,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
		DisableCompression:    true,
	}

	// 设置代理
	if d.ProxyURL != "" {
		log.Printf("正在配置代理: %s", d.ProxyURL)
		proxyURL, err := url.Parse(d.ProxyURL)
		if err != nil {
			log.Printf("代理URL解析失败: %v", err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("代理设置成功: %s", proxyURL.Host)
		}
	}

	// 测试代理连接
	if d.ProxyURL != "" {
		if err := d.testProxyConnection(); err != nil {
			log.Printf("警告: 代理连接测试失败: %v", err)
		} else {
			log.Printf("代理连接测试成功")
		}
	}

	if d.UseHTTP2 {
		http2.ConfigureTransport(transport)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // 不要跳过TLS验证
		MinVersion:         tls.VersionTLS12,
	}

	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
		Timeout:   time.Duration(d.Timeout) * time.Second,
	}
}

// testProxyConnection 测试代理连接
func (d *TileDownloader) testProxyConnection() error {
	// 创建一个简单的测试请求
	testURL := "http://httpbin.org/ip"

	transport := &http.Transport{}
	if d.ProxyURL != "" {
		proxyURL, err := url.Parse(d.ProxyURL)
		if err != nil {
			return err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// safeCloseResponse 安全关闭响应体
func safeCloseResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// recordError 记录错误
func (d *TileDownloader) recordError(err error) {
	if err == nil {
		return
	}

	errStr := err.Error()

	d.errorStatsMu.Lock()
	defer d.errorStatsMu.Unlock()

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

	d.errorStats[simplifiedErr]++
}

// printErrorStats 打印错误统计
func (d *TileDownloader) printErrorStats() {
	d.errorStatsMu.RLock()
	defer d.errorStatsMu.RUnlock()

	if len(d.errorStats) == 0 {
		return
	}

	log.Println("错误统计:")
	for err, count := range d.errorStats {
		log.Printf("  %s: %d次", err, count)
	}
}

// Init 初始化下载器
func (d *TileDownloader) Init() error {
	d.client = d.createHTTPClient()

	if d.RateLimit > 0 {
		d.limiter = rate.NewLimiter(rate.Limit(d.RateLimit), d.RateLimit)
		log.Printf("速率限制: %d 请求/秒", d.RateLimit)
	}

	d.workerPool = NewWorkerPool(d.Threads, d)
	log.Printf("并发线程数: %d", d.Threads)

	return nil
}

// Cleanup 清理资源
func (d *TileDownloader) Cleanup() {
	if d.client != nil {
		if transport, ok := d.client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}

	if err := d.saveResumeData(); err != nil {
		log.Printf("清理时保存断点数据失败: %v", err)
	}

	// 打印错误统计
	d.printErrorStats()
}

// doDownload 执行下载
func (d *TileDownloader) doDownload(task *DownloadTask) error {
	req, err := http.NewRequest("GET", task.URL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("User-Agent", d.UserAgent)
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	// 添加Referer头，有些地图服务需要
	req.Header.Set("Referer", "https://www.google.com/")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.Timeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer safeCloseResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, d.BufferSize)
	var data []byte
	var totalRead int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
			totalRead += int64(n)

			if totalRead > d.MaxFileSize {
				return fmt.Errorf("文件大小超过限制: %d > %d", totalRead, d.MaxFileSize)
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

	if int64(len(data)) < d.MinFileSize {
		return fmt.Errorf("文件太小: %d < %d", len(data), d.MinFileSize)
	}

	if !d.validateFileFormat(data) {
		return fmt.Errorf("无效的文件格式")
	}

	if err := os.MkdirAll(filepath.Dir(task.SavePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	if err := os.WriteFile(task.SavePath, data, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	var md5Hash string
	if d.CheckMD5 {
		hash := md5.Sum(data)
		md5Hash = hex.EncodeToString(hash[:])
	}

	d.markTileComplete(task.Tile, task.SavePath, int64(len(data)), md5Hash)

	return nil
}

// downloadTask 下载单个任务
func (d *TileDownloader) downloadTask(task *DownloadTask, stats *DownloadStats) {
	if d.limiter != nil {
		if err := d.limiter.Wait(context.Background()); err != nil {
			log.Printf("速率限制错误: %v", err)
			return
		}
	}

	if d.SkipExisting {
		if downloaded, info := d.isTileDownloaded(task.Tile); downloaded {
			atomic.AddInt64(&stats.Skipped, 1)
			if info != nil {
				atomic.AddInt64(&stats.BytesTotal, info.FileSize)
			}
			return
		}
	}

	var lastErr error
	maxAttempts := d.Retries + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond // 增加延迟
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
		d.recordError(err)

		errStr := err.Error()
		if strings.Contains(errStr, "404") ||
			strings.Contains(errStr, "403") ||
			strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "400") {
			// 这些错误重试没用
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
	d.markTileFailed(task.Tile, lastErr.Error())

	if atomic.LoadInt64(&stats.Failed)%10 == 0 { // 降低打印频率
		log.Printf("累计失败数: %d", atomic.LoadInt64(&stats.Failed))
	}
}

// Run 运行下载任务
func (d *TileDownloader) Run() error {
	defer d.Cleanup()

	if err := d.Init(); err != nil {
		return fmt.Errorf("初始化失败: %v", err)
	}

	if err := d.loadResumeData(); err != nil {
		return fmt.Errorf("加载断点数据失败: %v", err)
	}

	log.Printf("正在并行计算瓦片范围...")
	tiles, err := d.CalculateTiles()
	if err != nil {
		return fmt.Errorf("计算瓦片失败: %v", err)
	}

	if len(tiles) == 0 {
		return fmt.Errorf("没有找到范围内的瓦片")
	}

	if err := os.MkdirAll(d.SaveDir, 0755); err != nil {
		return fmt.Errorf("创建保存目录失败: %v", err)
	}

	d.stats = &DownloadStats{
		Total:        int64(len(tiles)),
		StartTime:    time.Now(),
		SpeedHistory: make([]SpeedRecord, 0, 100),
	}

	log.Printf("开始下载 %d 个瓦片", len(tiles))
	log.Printf("保存目录: %s", d.SaveDir)
	log.Printf("保存格式: %s", d.Format)
	log.Printf("HTTP/2: %v, Keep-Alive: %v", d.UseHTTP2, d.KeepAlive)
	if d.ProxyURL != "" {
		log.Printf("使用代理: %s", d.ProxyURL)
	}
	if d.RateLimit > 0 {
		log.Printf("速率限制: %d 请求/秒", d.RateLimit)
	}

	stopMonitor := make(chan bool)
	go d.monitorStats(stopMonitor)

	d.workerPool.Start(d.stats)

	d.submitTasksInBatches(tiles)

	d.workerPool.Stop()

	close(stopMonitor)

	time.Sleep(500 * time.Millisecond)

	d.printFinalStats()

	if d.stats.Failed > 0 {
		return fmt.Errorf("有 %d 个瓦片下载失败", d.stats.Failed)
	}

	return nil
}

// monitorStats 监控统计信息
func (d *TileDownloader) monitorStats(stop chan bool) {
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

			currentSuccess := atomic.LoadInt64(&d.stats.Success)
			currentBytes := atomic.LoadInt64(&d.stats.BytesTotal)
			currentFailed := atomic.LoadInt64(&d.stats.Failed)
			currentSkipped := atomic.LoadInt64(&d.stats.Skipped)

			successDiff := currentSuccess - lastSuccess
			bytesDiff := currentBytes - lastBytes

			var speed float64
			var countSpeed float64
			if duration > 0 {
				speed = float64(bytesDiff) / 1024 / duration
				countSpeed = float64(successDiff) / duration
			}

			d.stats.mu.Lock()
			d.stats.SpeedHistory = append(d.stats.SpeedHistory, SpeedRecord{
				Time:  now,
				Speed: speed,
				Count: int64(countSpeed),
			})

			if len(d.stats.SpeedHistory) > 100 {
				d.stats.SpeedHistory = d.stats.SpeedHistory[1:]
			}
			d.stats.mu.Unlock()

			totalProcessed := currentSuccess + currentSkipped
			percent := float64(totalProcessed) / float64(d.stats.Total) * 100

			activeWorkers := atomic.LoadInt32(&d.stats.ActiveWorkers)

			log.Printf("进度: %d/%d (%.1f%%) | 速度: %.1f KB/s, %.1f 瓦片/秒 | 活跃线程: %d | 成功: %d, 失败: %d, 跳过: %d",
				totalProcessed, d.stats.Total, percent, speed, countSpeed, activeWorkers,
				currentSuccess, currentFailed, currentSkipped)

			lastSuccess = currentSuccess
			lastBytes = currentBytes
			lastTime = now

			if totalProcessed >= d.stats.Total {
				return
			}
		case <-stop:
			return
		}
	}
}

// 辅助函数
func deg2num(lon, lat float64, zoom int) (x, y int) {
	n := 1 << zoom
	x = int((lon + 180.0) / 360.0 * float64(n))

	latRad := lat * math.Pi / 180.0
	y = int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * float64(n))

	return x, y
}

func (d *TileDownloader) validateFileFormat(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return true
	}

	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	if len(data) >= 12 && data[0] == 'R' && data[1] == 'I' && data[2] == 'F' &&
		data[3] == 'F' && data[8] == 'W' && data[9] == 'E' && data[10] == 'B' &&
		data[11] == 'P' {
		return true
	}

	content := string(data[:minInt(100, len(data))])
	if strings.Contains(content, "error") ||
		strings.Contains(content, "not found") ||
		strings.Contains(content, "forbidden") {
		return false
	}

	return int64(len(data)) > d.MinFileSize && int64(len(data)) < d.MaxFileSize
}

func (d *TileDownloader) GetTileURL(t Tile) string {
	url := d.URLTemplate
	url = strings.ReplaceAll(url, "{x}", strconv.Itoa(t.X))
	url = strings.ReplaceAll(url, "{y}", strconv.Itoa(t.Y))
	url = strings.ReplaceAll(url, "{z}", strconv.Itoa(t.Z))
	url = strings.ReplaceAll(url, "{-y}", strconv.Itoa((1<<t.Z)-t.Y-1))
	return url
}

func (d *TileDownloader) GetSavePath(t Tile) (string, error) {
	var path string

	switch d.Format {
	case "zxy":
		path = filepath.Join(d.SaveDir, strconv.Itoa(t.Z),
			strconv.Itoa(t.X), strconv.Itoa(t.Y))
	case "xyz":
		path = filepath.Join(d.SaveDir, strconv.Itoa(t.X),
			strconv.Itoa(t.Y), strconv.Itoa(t.Z))
	case "z/x/y":
		path = filepath.Join(d.SaveDir, fmt.Sprintf("%d/%d/%d", t.Z, t.X, t.Y))
	default:
		path = filepath.Join(d.SaveDir, strconv.Itoa(t.Z),
			strconv.Itoa(t.X), strconv.Itoa(t.Y))
	}

	ext := d.getFileExtension()
	if !strings.HasSuffix(path, ext) {
		path += ext
	}

	return path, nil
}

func (d *TileDownloader) getFileExtension() string {
	if d.OutputType != "auto" && d.OutputType != "" {
		return "." + strings.TrimPrefix(d.OutputType, ".")
	}

	url := d.URLTemplate
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".png"
	}

	return ext
}

func (d *TileDownloader) tileKey(t Tile) string {
	return fmt.Sprintf("%d/%d/%d", t.Z, t.X, t.Y)
}

func (d *TileDownloader) isTileDownloaded(t Tile) (bool, *TileInfo) {
	if d.ResumeData == nil {
		return false, nil
	}

	key := d.tileKey(t)
	if info, ok := d.ResumeData.Completed[key]; ok {
		if stat, err := os.Stat(info.FilePath); err == nil && stat.Size() == info.FileSize {
			return true, &info
		}
	}
	return false, nil
}

func (d *TileDownloader) markTileComplete(t Tile, filePath string, fileSize int64, md5Hash string) {
	if d.ResumeData == nil {
		return
	}

	d.ResumeMutex.Lock()
	defer d.ResumeMutex.Unlock()

	key := d.tileKey(t)
	d.ResumeData.Completed[key] = TileInfo{
		X:          t.X,
		Y:          t.Y,
		Z:          t.Z,
		FilePath:   filePath,
		FileSize:   fileSize,
		MD5:        md5Hash,
		Downloaded: time.Now(),
		Valid:      true,
	}

	delete(d.ResumeData.Failed, key)
}

func (d *TileDownloader) markTileFailed(t Tile, reason string) {
	if d.ResumeData == nil {
		return
	}

	d.ResumeMutex.Lock()
	defer d.ResumeMutex.Unlock()

	key := d.tileKey(t)
	d.ResumeData.Failed[key] = reason
}

func (d *TileDownloader) loadResumeData() error {
	if d.ResumeFile == "" {
		d.ResumeData = &ResumeData{
			Version:      "2.0",
			Completed:    make(map[string]TileInfo),
			Failed:       make(map[string]string),
			DownloadTime: time.Now(),
		}
		return nil
	}

	resumePath := filepath.Join(d.SaveDir, d.ResumeFile)
	if _, err := os.Stat(resumePath); os.IsNotExist(err) {
		d.ResumeData = &ResumeData{
			Version:      "2.0",
			Completed:    make(map[string]TileInfo),
			Failed:       make(map[string]string),
			DownloadTime: time.Now(),
		}
		return nil
	}

	data, err := os.ReadFile(resumePath)
	if err != nil {
		return fmt.Errorf("读取断点文件失败: %v", err)
	}

	var resumeData ResumeData
	if err := json.Unmarshal(data, &resumeData); err != nil {
		return fmt.Errorf("解析断点文件失败: %v", err)
	}

	d.ResumeData = &resumeData
	log.Printf("从 %s 加载断点数据，已下载 %d 个瓦片", resumePath, len(resumeData.Completed))

	return nil
}

func (d *TileDownloader) saveResumeData() error {
	if d.ResumeData == nil || d.ResumeFile == "" {
		return nil
	}

	d.ResumeMutex.Lock()
	defer d.ResumeMutex.Unlock()

	d.ResumeData.TotalTiles = int(d.stats.Total)
	d.ResumeData.DownloadTime = time.Now()

	resumePath := filepath.Join(d.SaveDir, d.ResumeFile)
	data, err := json.MarshalIndent(d.ResumeData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化断点数据失败: %v", err)
	}

	tmpPath := resumePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, resumePath)
}

func (d *TileDownloader) CalculateTiles() ([]Tile, error) {
	var tiles []Tile

	var mu sync.Mutex
	var wg sync.WaitGroup
	zoomChan := make(chan int, d.MaxZoom-d.MinZoom+1)

	workers := d.Threads
	if workers > d.MaxZoom-d.MinZoom+1 {
		workers = d.MaxZoom - d.MinZoom + 1
	}
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for z := range zoomChan {
				minX, maxY := deg2num(d.MinLon, d.MaxLat, z)
				maxX, minY := deg2num(d.MaxLon, d.MinLat, z)

				maxTile := 1 << z
				minX = maxInt(0, minInt(minX, maxTile-1))
				maxX = maxInt(0, minInt(maxX, maxTile-1))
				minY = maxInt(0, minInt(minY, maxTile-1))
				maxY = maxInt(0, minInt(maxY, maxTile-1))

				temp := minY
				minY = maxY
				maxY = temp

				var zoomTiles []Tile
				for x := minX; x <= maxX; x++ {
					for y := minY; y <= maxY; y++ {
						zoomTiles = append(zoomTiles, Tile{X: x, Y: y, Z: z})
					}
				}

				mu.Lock()
				tiles = append(tiles, zoomTiles...)
				mu.Unlock()
			}
		}()
	}

	for z := d.MinZoom; z <= d.MaxZoom; z++ {
		zoomChan <- z
	}
	close(zoomChan)
	wg.Wait()

	return tiles, nil
}

func (d *TileDownloader) submitTasksInBatches(tiles []Tile) {
	batchSize := d.BatchSize
	if batchSize > 200 {
		batchSize = 200
	}
	if batchSize < 1 {
		batchSize = 50
	}

	for i := 0; i < len(tiles); i += batchSize {
		end := i + batchSize
		if end > len(tiles) {
			end = len(tiles)
		}
		batch := tiles[i:end]

		for _, tile := range batch {
			if d.SkipExisting {
				if downloaded, _ := d.isTileDownloaded(tile); downloaded {
					atomic.AddInt64(&d.stats.Skipped, 1)
					continue
				}
			}

			url := d.GetTileURL(tile)
			savePath, err := d.GetSavePath(tile)
			if err != nil {
				log.Printf("获取保存路径失败: %v", err)
				continue
			}

			task := &DownloadTask{
				Tile:     tile,
				URL:      url,
				SavePath: savePath,
				Retry:    0,
			}

			d.workerPool.SubmitTask(task)
		}

		if i%1000 == 0 {
			submitted := i + batchSize
			if submitted > len(tiles) {
				submitted = len(tiles)
			}
			percent := float64(submitted) / float64(len(tiles)) * 100
			log.Printf("任务提交进度: %d/%d (%.1f%%)", submitted, len(tiles), percent)
		}
	}

	log.Printf("所有任务已提交到队列")
}

func (d *TileDownloader) printFinalStats() {
	duration := time.Since(d.stats.StartTime)
	totalProcessed := d.stats.Success + d.stats.Skipped

	log.Println(strings.Repeat("=", 60))
	log.Printf("下载完成!")
	log.Println(strings.Repeat("=", 60))
	log.Printf("总瓦片数: %d", d.stats.Total)
	log.Printf("成功下载: %d", d.stats.Success)
	log.Printf("跳过下载: %d", d.stats.Skipped)
	log.Printf("下载失败: %d", d.stats.Failed)
	log.Printf("重试次数: %d", d.stats.Retries)
	log.Printf("下载总量: %.2f MB", float64(d.stats.BytesTotal)/1024/1024)
	log.Printf("总用时: %v", duration)

	if duration.Seconds() > 0 {
		log.Printf("平均速度: %.1f 瓦片/秒", float64(totalProcessed)/duration.Seconds())
		log.Printf("下载速度: %.2f MB/秒", float64(d.stats.BytesTotal)/1024/1024/duration.Seconds())
	}

	log.Println(strings.Repeat("=", 60))

	if err := d.generateReport(); err != nil {
		log.Printf("生成报告失败: %v", err)
	}
}

func (d *TileDownloader) generateReport() error {
	reportPath := filepath.Join(d.SaveDir, "download_report.json")

	report := map[string]interface{}{
		"download_time":    time.Now().Format("2006-01-02 15:04:05"),
		"url_template":     d.URLTemplate,
		"min_lon":          d.MinLon,
		"max_lon":          d.MaxLon,
		"min_lat":          d.MinLat,
		"max_lat":          d.MaxLat,
		"min_zoom":         d.MinZoom,
		"max_zoom":         d.MaxZoom,
		"save_dir":         d.SaveDir,
		"format":           d.Format,
		"threads":          d.Threads,
		"total_tiles":      d.stats.Total,
		"success":          d.stats.Success,
		"failed":           d.stats.Failed,
		"skipped":          d.stats.Skipped,
		"retries":          d.stats.Retries,
		"bytes_total":      d.stats.BytesTotal,
		"duration_seconds": time.Since(d.stats.StartTime).Seconds(),
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(reportPath, data, 0644)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewWorkerPool 创建工作池
func NewWorkerPool(workers int, downloader *TileDownloader) *WorkerPool {
	return &WorkerPool{
		workers:    workers,
		taskQueue:  make(chan *DownloadTask, workers*2),
		downloader: downloader,
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start(stats *DownloadStats) {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(stats)
	}
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	close(wp.taskQueue)
	wp.wg.Wait()
}

// worker 工作协程
func (wp *WorkerPool) worker(stats *DownloadStats) {
	defer wp.wg.Done()

	for task := range wp.taskQueue {
		atomic.AddInt32(&stats.ActiveWorkers, 1)
		wp.downloader.downloadTask(task, stats)
		atomic.AddInt32(&stats.ActiveWorkers, -1)
	}
}

// SubmitTask 提交任务
func (wp *WorkerPool) SubmitTask(task *DownloadTask) {
	wp.taskQueue <- task
}

// 主函数
func main() {
	urlTemplate := flag.String("url", "http://tile.openstreetmap.org/{z}/{x}/{y}.png", "图源URL模板")
	minLon := flag.Float64("min-lon", 116.0, "最小经度")
	minLat := flag.Float64("min-lat", 39.0, "最小纬度")
	maxLon := flag.Float64("max-lon", 117.0, "最大经度")
	maxLat := flag.Float64("max-lat", 40.0, "最大纬度")
	minZoom := flag.Int("min-zoom", 10, "最小层级")
	maxZoom := flag.Int("max-zoom", 12, "最大层级")
	saveDir := flag.String("save-dir", "./tiles", "保存目录")
	format := flag.String("format", "zxy", "保存格式")
	threads := flag.Int("threads", 10, "并发线程数")
	timeout := flag.Int("timeout", 60, "超时时间(秒)")
	retries := flag.Int("retries", 3, "重试次数")
	userAgent := flag.String("user-agent", "", "自定义User-Agent")
	outputType := flag.String("type", "auto", "输出文件类型")
	proxyURL := flag.String("proxy", "", "代理URL")
	resumeFile := flag.String("resume-file", ".resume.json", "断点续传记录文件")
	skipExisting := flag.Bool("skip-existing", true, "跳过已存在的文件")
	checkMD5 := flag.Bool("check-md5", false, "校验文件MD5")
	minFileSize := flag.Int64("min-size", 100, "最小文件大小(字节)")
	maxFileSize := flag.Int64("max-size", 2097152, "最大文件大小(字节)")
	rateLimit := flag.Int("rate-limit", 10, "每秒请求限制")
	batchSize := flag.Int("batch-size", 100, "批量下载大小")
	useHTTP2 := flag.Bool("http2", true, "使用HTTP/2")
	keepAlive := flag.Bool("keep-alive", true, "保持长连接")
	bufferSize := flag.Int("buffer-size", 65536, "缓冲区大小(字节)")
	force := flag.Bool("force", false, "强制重新下载")

	flag.Parse()

	downloader := NewTileDownloader()
	downloader.URLTemplate = *urlTemplate
	downloader.MinLon = *minLon
	downloader.MinLat = *minLat
	downloader.MaxLon = *maxLon
	downloader.MaxLat = *maxLat
	downloader.MinZoom = *minZoom
	downloader.MaxZoom = *maxZoom
	downloader.SaveDir = *saveDir
	downloader.Format = *format
	downloader.Threads = *threads
	downloader.Timeout = *timeout
	downloader.Retries = *retries
	downloader.OutputType = *outputType
	downloader.ProxyURL = *proxyURL
	downloader.ResumeFile = *resumeFile
	downloader.SkipExisting = *skipExisting && !*force
	downloader.CheckMD5 = *checkMD5
	downloader.MinFileSize = *minFileSize
	downloader.MaxFileSize = *maxFileSize
	downloader.RateLimit = *rateLimit
	downloader.BatchSize = *batchSize
	downloader.UseHTTP2 = *useHTTP2
	downloader.KeepAlive = *keepAlive
	downloader.BufferSize = *bufferSize

	if *userAgent != "" {
		downloader.UserAgent = *userAgent
	}

	if err := downloader.Run(); err != nil {
		log.Fatalf("下载失败: %v", err)
	}
}

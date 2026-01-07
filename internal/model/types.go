// Package model 定义数据模型
package model

import "time"

// Tile 瓦片坐标
type Tile struct {
	X, Y, Z int
}

// DownloadTask 下载任务
type DownloadTask struct {
	Tile     Tile
	URL      string
	SavePath string
	Retry    int
	Priority int
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

// Config 下载器配置
type Config struct {
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
}

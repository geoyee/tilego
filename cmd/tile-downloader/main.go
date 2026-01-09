package main

import (
	"flag"
	"log"
	"math"

	"github.com/geoyee/tilego/internal/download"
	"github.com/geoyee/tilego/internal/model"
)

func main() {
	config := &model.Config{}

	flag.StringVar(&config.URLTemplate, "url", "", "[Required] Tile URL template (e.g., https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x})")
	flag.Float64Var(&config.MinLon, "min-lon", math.NaN(), "[Required] Minimum longitude (e.g., -180.0)")
	flag.Float64Var(&config.MinLat, "min-lat", math.NaN(), "[Required] Minimum latitude (e.g., -90.0)")
	flag.Float64Var(&config.MaxLon, "max-lon", math.NaN(), "[Required] Maximum longitude (e.g., 180.0)")
	flag.Float64Var(&config.MaxLat, "max-lat", math.NaN(), "[Required] Maximum latitude (e.g., 90.0)")
	flag.IntVar(&config.MinZoom, "min-zoom", 0, "[Optional] Minimum zoom level")
	flag.IntVar(&config.MaxZoom, "max-zoom", 18, "[Optional] Maximum zoom level")
	flag.StringVar(&config.SaveDir, "dir", "./tiles", "[Optional] Save directory")
	flag.StringVar(&config.Format, "format", "zxy", "[Optional] Save format (e.g., zxy, xyz, z/x/y)")
	flag.IntVar(&config.Threads, "threads", 10, "[Optional] Number of concurrent threads")
	flag.IntVar(&config.Timeout, "timeout", 60, "[Optional] Timeout in seconds")
	flag.IntVar(&config.Retries, "retries", 5, "[Optional] Number of retries")
	flag.StringVar(&config.ProxyURL, "proxy", "", "[Optional] Proxy URL (e.g., http://127.0.0.1:7890)")
	flag.StringVar(&config.UserAgent, "user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "[Optional] User agent")
	flag.BoolVar(&config.SkipExisting, "skip", true, "[Optional] Skip existing files")
	flag.BoolVar(&config.CheckMD5, "md5", false, "[Optional] Verify file MD5")
	flag.Int64Var(&config.MinFileSize, "min-size", 100, "[Optional] Minimum file size in bytes")
	flag.Int64Var(&config.MaxFileSize, "max-size", 2097152, "[Optional] Maximum file size in bytes")
	flag.IntVar(&config.RateLimit, "rate", 10, "[Optional] Rate limit in requests/second")
	flag.BoolVar(&config.UseHTTP2, "http2", true, "[Optional] Enable HTTP/2")
	flag.BoolVar(&config.KeepAlive, "keep-alive", true, "[Optional] Enable persistent connections")
	flag.IntVar(&config.BatchSize, "batch", 1000, "[Optional] Batch processing size")
	flag.IntVar(&config.BufferSize, "buffer", 8192, "[Optional] Download buffer size")
	flag.StringVar(&config.ResumeFile, "resume", ".tilego-resume.json", "[Optional] Resume file name")

	flag.Parse()

	if config.URLTemplate == "" {
		log.Fatal("Error: -url parameter is required")
	}
	if math.IsNaN(config.MinLon) {
		log.Fatal("Error: -min-lon parameter is required")
	}
	if math.IsNaN(config.MaxLon) {
		log.Fatal("Error: -max-lon parameter is required")
	}
	if math.IsNaN(config.MinLat) {
		log.Fatal("Error: -min-lat parameter is required")
	}
	if math.IsNaN(config.MaxLat) {
		log.Fatal("Error: -max-lat parameter is required")
	}

	downloader := download.NewDownloader(config)

	log.Println("========================================")
	log.Println("tilego - Map Tile Downloader")
	log.Println("========================================")
	log.Printf("Tile source: %s", config.URLTemplate)
	log.Printf("Download range: %.6f,%.6f - %.6f,%.6f", config.MinLon, config.MinLat, config.MaxLon, config.MaxLat)
	log.Printf("Zoom levels: %d - %d", config.MinZoom, config.MaxZoom)
	log.Printf("Save directory: %s", config.SaveDir)
	log.Printf("Concurrent threads: %d", config.Threads)
	log.Println("========================================")

	if err := downloader.Run(); err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	log.Println("Download task completed successfully")
}

// Package main 瓦片下载器主程序
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

	// 命令行参数解析
	flag.StringVar(&config.URLTemplate, "url", "", "[必填] 图源URL模板 (如 https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x})")
	flag.Float64Var(&config.MinLon, "min-lon", math.NaN(), "[必填] 最小经度 (如 -180.0)")
	flag.Float64Var(&config.MinLat, "min-lat", math.NaN(), "[必填] 最小纬度 (如 -90.0)")
	flag.Float64Var(&config.MaxLon, "max-lon", math.NaN(), "[必填] 最大经度 (如 180.0)")
	flag.Float64Var(&config.MaxLat, "max-lat", math.NaN(), "[必填] 最大纬度 (如 90.0)")
	flag.IntVar(&config.MinZoom, "min-zoom", 0, "[选填] 最小缩放级别")
	flag.IntVar(&config.MaxZoom, "max-zoom", 18, "[选填] 最大缩放级别")
	flag.StringVar(&config.SaveDir, "dir", "./tiles", "[选填] 保存目录")
	flag.StringVar(&config.Format, "format", "zxy", "[选填] 保存格式 (可选 [xyz][z/x/y])")
	flag.IntVar(&config.Threads, "threads", 10, "[选填] 并发线程数")
	flag.IntVar(&config.Timeout, "timeout", 60, "[选填] 超时时间@秒")
	flag.IntVar(&config.Retries, "retries", 5, "[选填] 重试次数")
	flag.StringVar(&config.ProxyURL, "proxy", "", "[选填] 代理URL (如 http://127.0.0.1:7890)")
	flag.StringVar(&config.UserAgent, "user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "[选填] 用户代理")
	flag.BoolVar(&config.SkipExisting, "skip", true, "[选填] 是否跳过已存在的文件")
	flag.BoolVar(&config.CheckMD5, "md5", false, "[选填] 是否校验文件MD5")
	flag.Int64Var(&config.MinFileSize, "min-size", 100, "[选填] 最小文件大小@字节")
	flag.Int64Var(&config.MaxFileSize, "max-size", 2097152, "[选填] 最大文件大小@字节") // 2*1024*1024
	flag.IntVar(&config.RateLimit, "rate", 10, "[选填] 速率限制@请求/秒")
	flag.BoolVar(&config.UseHTTP2, "http2", true, "[选填] 是否启用HTTP/2")
	flag.BoolVar(&config.KeepAlive, "keep-alive", true, "[选填] 是否启用长连接")

	flag.Parse()

	// 校验必填参数
	if config.URLTemplate == "" {
		log.Fatal("错误: 必须指定 -url 参数")
	}
	if math.IsNaN(config.MinLon) {
		log.Fatal("错误: 必须指定 -min-lon 参数")
	}
	if math.IsNaN(config.MaxLon) {
		log.Fatal("错误: 必须指定 -max-lon 参数")
	}
	if math.IsNaN(config.MinLat) {
		log.Fatal("错误: 必须指定 -min-lat 参数")
	}
	if math.IsNaN(config.MaxLat) {
		log.Fatal("错误: 必须指定 -max-lat 参数")
	}

	// 创建下载器
	downloader := download.NewDownloader(config)

	// 启动下载
	log.Println("========================================")
	log.Println("tilego")
	log.Println("========================================")
	log.Printf("瓦片图源: %s", config.URLTemplate)
	log.Printf("下载范围: %.6f,%.6f - %.6f,%.6f", config.MinLon, config.MinLat, config.MaxLon, config.MaxLat)
	log.Printf("缩放级别: %d - %d", config.MinZoom, config.MaxZoom)
	log.Println("========================================")

	if err := downloader.Run(); err != nil {
		log.Fatalf("下载失败: %v", err)
	}

	log.Println("下载任务全部完成")
}

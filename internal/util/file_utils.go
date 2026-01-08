// Package util 提供工具函数
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GetFileExtension 获取文件扩展名
func GetFileExtension(urlTemplate, outputType string) string {
	if outputType != "auto" && outputType != "" {
		return "." + strings.TrimPrefix(outputType, ".")
	}

	url := urlTemplate
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".png"
	}

	return ext
}

// ValidateFileFormat 验证文件格式
func ValidateFileFormat(data []byte, minFileSize, maxFileSize int64) bool {
	if len(data) < 8 {
		return false
	}

	// PNG 校验
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return true
	}

	// JPG 校验
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}

	// WEBP 校验
	if len(data) >= 12 && data[0] == 'R' && data[1] == 'I' && data[2] == 'F' &&
		data[3] == 'F' && data[8] == 'W' && data[9] == 'E' && data[10] == 'B' &&
		data[11] == 'P' {
		return true
	}

	// 排除错误内容
	content := string(data[:MinInt(100, len(data))])
	if strings.Contains(content, "error") ||
		strings.Contains(content, "not found") ||
		strings.Contains(content, "forbidden") {
		return false
	}

	return int64(len(data)) >= minFileSize && int64(len(data)) <= maxFileSize
}

// GetTileURL 获取瓦片URL
func GetTileURL(urlTemplate string, x, y, z int) string {
	url := urlTemplate
	url = strings.ReplaceAll(url, "{x}", strconv.Itoa(x))
	url = strings.ReplaceAll(url, "{y}", strconv.Itoa(y))
	url = strings.ReplaceAll(url, "{z}", strconv.Itoa(z))
	url = strings.ReplaceAll(url, "{-y}", strconv.Itoa((1<<z)-y-1))
	return url
}

// GetSavePath 获取保存路径
func GetSavePath(saveDir, format string, x, y, z int, ext string) (string, error) {
	var path string

	switch format {
	case "zxy":
		path = filepath.Join(saveDir, strconv.Itoa(z),
			strconv.Itoa(x), strconv.Itoa(y))
	case "xyz":
		path = filepath.Join(saveDir, strconv.Itoa(x),
			strconv.Itoa(y), strconv.Itoa(z))
	case "z/x/y":
		path = filepath.Join(saveDir, fmt.Sprintf("%d/%d/%d", z, x, y))
	default:
		path = filepath.Join(saveDir, strconv.Itoa(z),
			strconv.Itoa(x), strconv.Itoa(y))
	}

	if !strings.HasSuffix(path, ext) {
		path += ext
	}

	return path, nil
}

// EnsureDirExists 确保目录存在
func EnsureDirExists(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// MinInt 最小值函数
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GenerateTileKey 生成瓦片唯一标识
func GenerateTileKey(z, x, y int) string {
	return fmt.Sprintf("%d/%d/%d", z, x, y)
}

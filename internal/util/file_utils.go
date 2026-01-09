// Package util provides utility functions for the tile downloader.
//
// This package contains helper functions for file operations, URL handling,
// and tile coordinate processing.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultFileExtension = ".png"
	maxErrorContentLen   = 100
)

func GetFileExtension(urlTemplate, outputType string) string {
	if outputType != "auto" && outputType != "" {
		return "." + strings.TrimPrefix(outputType, ".")
	}

	url := urlTemplate
	if idx := strings.IndexByte(url, '?'); idx != -1 {
		url = url[:idx]
	}

	ext := filepath.Ext(url)
	if ext == "" {
		ext = defaultFileExtension
	}

	return ext
}

func ValidateFileFormat(data []byte, minFileSize, maxFileSize int64) bool {
	if len(data) < 8 {
		return false
	}

	if isValidPNG(data) || isValidJPG(data) || isValidWebP(data) {
		return true
	}

	if containsErrorContent(data) {
		return false
	}

	fileSize := int64(len(data))
	return fileSize >= minFileSize && fileSize <= maxFileSize
}

func isValidPNG(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0x89 &&
		data[1] == 'P' &&
		data[2] == 'N' &&
		data[3] == 'G'
}

func isValidJPG(data []byte) bool {
	return len(data) >= 3 &&
		data[0] == 0xFF &&
		data[1] == 0xD8 &&
		data[2] == 0xFF
}

func isValidWebP(data []byte) bool {
	return len(data) >= 12 &&
		data[0] == 'R' &&
		data[1] == 'I' &&
		data[2] == 'F' &&
		data[3] == 'F' &&
		data[8] == 'W' &&
		data[9] == 'E' &&
		data[10] == 'B' &&
		data[11] == 'P'
}

func containsErrorContent(data []byte) bool {
	contentLen := MinInt(maxErrorContentLen, len(data))
	content := string(data[:contentLen])

	return strings.Contains(content, "error") ||
		strings.Contains(content, "not found") ||
		strings.Contains(content, "forbidden")
}

func GetTileURL(urlTemplate string, x, y, z int) string {
	var builder strings.Builder
	builder.Grow(len(urlTemplate) + 30)

	parts := []struct {
		placeholder string
		value       int
	}{
		{"{x}", x},
		{"{y}", y},
		{"{z}", z},
		{"{-y}", (1 << z) - y - 1},
	}

	lastIndex := 0
	for _, part := range parts {
		for {
			idx := strings.Index(urlTemplate[lastIndex:], part.placeholder)
			if idx == -1 {
				break
			}
			actualIdx := lastIndex + idx
			builder.WriteString(urlTemplate[lastIndex:actualIdx])
			builder.WriteString(strconv.Itoa(part.value))
			lastIndex = actualIdx + len(part.placeholder)
		}
	}
	builder.WriteString(urlTemplate[lastIndex:])

	return builder.String()
}

func GetSavePath(saveDir, format string, x, y, z int, ext string) (string, error) {
	var path string

	switch format {
	case "zxy":
		path = filepath.Join(saveDir, strconv.Itoa(z), strconv.Itoa(x), strconv.Itoa(y))
	case "xyz":
		path = filepath.Join(saveDir, strconv.Itoa(x), strconv.Itoa(y), strconv.Itoa(z))
	case "z/x/y":
		path = filepath.Join(saveDir, fmt.Sprintf("%d/%d/%d", z, x, y))
	default:
		path = filepath.Join(saveDir, strconv.Itoa(z), strconv.Itoa(x), strconv.Itoa(y))
	}

	if !strings.HasSuffix(path, ext) {
		path += ext
	}

	return path, nil
}

func EnsureDirExists(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GenerateTileKey(z, x, y int) string {
	return fmt.Sprintf("%d/%d/%d", z, x, y)
}

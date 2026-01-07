// Package resume 提供断点续传功能
package resume

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/geoyee/tilego/internal/model"
	"github.com/geoyee/tilego/internal/util"
)

// ResumeManager 断点续传管理器
type ResumeManager struct {
	resumeData *model.ResumeData
	resumeFile string
	saveDir    string
	mu         sync.RWMutex
}

// NewResumeManager 创建断点续传管理器
func NewResumeManager(saveDir, resumeFile string) *ResumeManager {
	return &ResumeManager{
		resumeFile: resumeFile,
		saveDir:    saveDir,
	}
}

// LoadResumeData 加载断点续传数据
func (rm *ResumeManager) LoadResumeData() error {
	if rm.resumeFile == "" {
		rm.resumeData = &model.ResumeData{
			Version:      "2.0",
			Completed:    make(map[string]model.TileInfo),
			Failed:       make(map[string]string),
			DownloadTime: time.Now(),
		}
		return nil
	}

	resumePath := filepath.Join(rm.saveDir, rm.resumeFile)
	if _, err := os.Stat(resumePath); os.IsNotExist(err) {
		rm.resumeData = &model.ResumeData{
			Version:      "2.0",
			Completed:    make(map[string]model.TileInfo),
			Failed:       make(map[string]string),
			DownloadTime: time.Now(),
		}
		return nil
	}

	data, err := os.ReadFile(resumePath)
	if err != nil {
		return fmt.Errorf("读取断点文件失败: %v", err)
	}

	var resumeData model.ResumeData
	if err := json.Unmarshal(data, &resumeData); err != nil {
		return fmt.Errorf("解析断点文件失败: %v", err)
	}

	rm.resumeData = &resumeData
	log.Printf("加载断点数据成功: 已完成 %d 个瓦片, 失败 %d 个瓦片",
		len(resumeData.Completed), len(resumeData.Failed))

	return nil
}

// SaveResumeData 保存断点续传数据
func (rm *ResumeManager) SaveResumeData(urlTemplate, format string, totalTiles int) error {
	if rm.resumeData == nil || rm.resumeFile == "" {
		return nil
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.resumeData.Version = "2.0"
	rm.resumeData.URLTemplate = urlTemplate
	rm.resumeData.SaveDir = rm.saveDir
	rm.resumeData.Format = format
	rm.resumeData.TotalTiles = totalTiles
	rm.resumeData.DownloadTime = time.Now()

	resumePath := filepath.Join(rm.saveDir, rm.resumeFile)
	data, err := json.MarshalIndent(rm.resumeData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化断点数据失败: %v", err)
	}

	if err := os.WriteFile(resumePath, data, 0644); err != nil {
		return fmt.Errorf("写入断点文件失败: %v", err)
	}

	return nil
}

// IsTileDownloaded 检查瓦片是否已下载
func (rm *ResumeManager) IsTileDownloaded(tile model.Tile) (bool, *model.TileInfo) {
	if rm.resumeData == nil {
		return false, nil
	}

	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	if info, ok := rm.resumeData.Completed[key]; ok {
		if stat, err := os.Stat(info.FilePath); err == nil && stat.Size() == info.FileSize {
			return true, &info
		}
		delete(rm.resumeData.Completed, key)
	}
	return false, nil
}

// MarkTileComplete 标记瓦片下载完成
func (rm *ResumeManager) MarkTileComplete(tile model.Tile, filePath string, fileSize int64, md5Hash string) {
	if rm.resumeData == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	rm.resumeData.Completed[key] = model.TileInfo{
		X:          tile.X,
		Y:          tile.Y,
		Z:          tile.Z,
		FilePath:   filePath,
		FileSize:   fileSize,
		MD5:        md5Hash,
		Downloaded: time.Now(),
		Valid:      true,
	}

	delete(rm.resumeData.Failed, key)
}

// MarkTileFailed 标记瓦片下载失败
func (rm *ResumeManager) MarkTileFailed(tile model.Tile, reason string) {
	if rm.resumeData == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	rm.resumeData.Failed[key] = reason
}

// GetResumeData 获取断点续传数据
func (rm *ResumeManager) GetResumeData() *model.ResumeData {
	return rm.resumeData
}

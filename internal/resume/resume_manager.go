// Package resume provides resume/continuation functionality for downloads.
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

type ResumeManager struct {
	resumeData *model.ResumeData
	resumeFile string
	saveDir    string
	mu         sync.RWMutex
}

func NewResumeManager(saveDir, resumeFile string) *ResumeManager {
	return &ResumeManager{
		resumeFile: resumeFile,
		saveDir:    saveDir,
	}
}

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
		return fmt.Errorf("failed to read resume file: %w", err)
	}

	var resumeData model.ResumeData
	if err := json.Unmarshal(data, &resumeData); err != nil {
		return fmt.Errorf("failed to parse resume file: %w", err)
	}

	rm.resumeData = &resumeData
	log.Printf("Resume data loaded: %d tiles completed, %d tiles failed",
		len(resumeData.Completed), len(resumeData.Failed))

	return nil
}

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
		return fmt.Errorf("failed to serialize resume data: %w", err)
	}

	if err := os.WriteFile(resumePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write resume file: %w", err)
	}

	return nil
}

func (rm *ResumeManager) IsTileDownloaded(tile model.Tile) (bool, *model.TileInfo) {
	rm.mu.RLock()

	if rm.resumeData == nil {
		rm.mu.RUnlock()
		return false, nil
	}

	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	info, ok := rm.resumeData.Completed[key]
	rm.mu.RUnlock()

	if !ok {
		return false, nil
	}

	stat, err := os.Stat(info.FilePath)
	if err == nil && stat.Size() == info.FileSize {
		return true, &info
	}

	rm.mu.Lock()
	delete(rm.resumeData.Completed, key)
	rm.mu.Unlock()

	return false, nil
}

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

func (rm *ResumeManager) MarkTileFailed(tile model.Tile, reason string) {
	if rm.resumeData == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	rm.resumeData.Failed[key] = reason
}

func (rm *ResumeManager) GetResumeData() *model.ResumeData {
	return rm.resumeData
}

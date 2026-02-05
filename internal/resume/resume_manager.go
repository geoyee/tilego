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
	ResumeData *model.ResumeData
	ResumeFile string
	SaveDir    string
	Mu         sync.RWMutex
}

func NewResumeManager(saveDir, resumeFile string) *ResumeManager {
	return &ResumeManager{
		ResumeFile: resumeFile,
		SaveDir:    saveDir,
	}
}

func (rm *ResumeManager) LoadResumeData() error {
	if rm.ResumeFile == "" {
		rm.ResumeData = &model.ResumeData{
			Version:      "2.0",
			Completed:    make(map[string]model.TileInfo),
			Failed:       make(map[string]string),
			DownloadTime: time.Now(),
		}
		return nil
	}
	resumePath := filepath.Join(rm.SaveDir, rm.ResumeFile)
	if _, err := os.Stat(resumePath); os.IsNotExist(err) {
		rm.ResumeData = &model.ResumeData{
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
	rm.ResumeData = &resumeData
	log.Printf("Resume data loaded: %d tiles completed, %d tiles failed",
		len(resumeData.Completed), len(resumeData.Failed))
	return nil
}

func (rm *ResumeManager) SaveResumeData(urlTemplate, format string, totalTiles int) error {
	if rm.ResumeData == nil || rm.ResumeFile == "" {
		return nil
	}
	rm.Mu.Lock()
	defer rm.Mu.Unlock()
	rm.ResumeData.Version = "2.0"
	rm.ResumeData.URLTemplate = urlTemplate
	rm.ResumeData.SaveDir = rm.SaveDir
	rm.ResumeData.Format = format
	rm.ResumeData.TotalTiles = totalTiles
	rm.ResumeData.DownloadTime = time.Now()
	resumePath := filepath.Join(rm.SaveDir, rm.ResumeFile)
	data, err := json.MarshalIndent(rm.ResumeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize resume data: %w", err)
	}
	if err := os.WriteFile(resumePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write resume file: %w", err)
	}
	return nil
}

func (rm *ResumeManager) IsTileDownloaded(tile model.Tile) (bool, *model.TileInfo) {
	rm.Mu.RLock()
	if rm.ResumeData == nil {
		rm.Mu.RUnlock()
		return false, nil
	}
	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	info, ok := rm.ResumeData.Completed[key]
	rm.Mu.RUnlock()
	if !ok {
		return false, nil
	}
	stat, err := os.Stat(info.FilePath)
	if err == nil && stat.Size() == info.FileSize {
		return true, &info
	}
	rm.Mu.Lock()
	delete(rm.ResumeData.Completed, key)
	rm.Mu.Unlock()
	return false, nil
}

func (rm *ResumeManager) MarkTileComplete(tile model.Tile, filePath string, fileSize int64, md5Hash string) {
	if rm.ResumeData == nil {
		return
	}
	rm.Mu.Lock()
	defer rm.Mu.Unlock()
	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	rm.ResumeData.Completed[key] = model.TileInfo{
		X:          tile.X,
		Y:          tile.Y,
		Z:          tile.Z,
		FilePath:   filePath,
		FileSize:   fileSize,
		MD5:        md5Hash,
		Downloaded: time.Now(),
		Valid:      true,
	}
	delete(rm.ResumeData.Failed, key)
}

func (rm *ResumeManager) MarkTileFailed(tile model.Tile, reason string) {
	if rm.ResumeData == nil {
		return
	}
	rm.Mu.Lock()
	defer rm.Mu.Unlock()
	key := util.GenerateTileKey(tile.Z, tile.X, tile.Y)
	rm.ResumeData.Failed[key] = reason
}

func (rm *ResumeManager) GetResumeData() *model.ResumeData {
	return rm.ResumeData
}

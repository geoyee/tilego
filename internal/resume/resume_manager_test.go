package resume

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/geoyee/tilego/internal/model"
)

func TestNewResumeManager(t *testing.T) {
	rm := NewResumeManager("/tmp/tiles", "resume.json")
	if rm == nil {
		t.Fatal("NewResumeManager returned nil")
	}

	if rm.SaveDir != "/tmp/tiles" {
		t.Errorf("Expected SaveDir=/tmp/tiles, got %s", rm.SaveDir)
	}

	if rm.ResumeFile != "resume.json" {
		t.Errorf("Expected ResumeFile=resume.json, got %s", rm.ResumeFile)
	}
}

func TestLoadResumeDataEmpty(t *testing.T) {
	rm := NewResumeManager("/tmp/tiles", "")
	err := rm.LoadResumeData()
	if err != nil {
		t.Fatalf("LoadResumeData failed: %v", err)
	}

	if rm.ResumeData == nil {
		t.Fatal("ResumeData is nil")
	}

	if rm.ResumeData.Version != "1.0" {
		t.Errorf("Expected Version=1.0, got %s", rm.ResumeData.Version)
	}

	if rm.ResumeData.Completed == nil {
		t.Error("Completed map should not be nil")
	}

	if rm.ResumeData.Failed == nil {
		t.Error("Failed map should not be nil")
	}
}

func TestLoadResumeDataNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	rm := NewResumeManager(tempDir, "nonexistent.json")

	err := rm.LoadResumeData()
	if err != nil {
		t.Fatalf("LoadResumeData failed: %v", err)
	}

	if rm.ResumeData == nil {
		t.Fatal("ResumeData is nil")
	}

	if len(rm.ResumeData.Completed) != 0 {
		t.Errorf("Expected empty Completed map, got %d items", len(rm.ResumeData.Completed))
	}
}

func TestSaveAndLoadResumeData(t *testing.T) {
	tempDir := t.TempDir()
	rm := NewResumeManager(tempDir, "resume.json")

	err := rm.LoadResumeData()
	if err != nil {
		t.Fatalf("LoadResumeData failed: %v", err)
	}

	rm.ResumeData.Completed["10/100/200"] = model.TileInfo{
		X:          100,
		Y:          200,
		Z:          10,
		FilePath:   filepath.Join(tempDir, "10/100/200.png"),
		FileSize:   1024,
		Downloaded: time.Now(),
		Valid:      true,
	}

	err = rm.SaveResumeData("https://example.com/{z}/{x}/{y}.png", "zxy", 1000)
	if err != nil {
		t.Fatalf("SaveResumeData failed: %v", err)
	}

	rm2 := NewResumeManager(tempDir, "resume.json")
	err = rm2.LoadResumeData()
	if err != nil {
		t.Fatalf("LoadResumeData failed: %v", err)
	}

	if len(rm2.ResumeData.Completed) != 1 {
		t.Errorf("Expected 1 completed tile, got %d", len(rm2.ResumeData.Completed))
	}

	if rm2.ResumeData.URLTemplate != "https://example.com/{z}/{x}/{y}.png" {
		t.Errorf("URLTemplate not loaded correctly")
	}

	if rm2.ResumeData.TotalTiles != 1000 {
		t.Errorf("Expected TotalTiles=1000, got %d", rm2.ResumeData.TotalTiles)
	}
}

func TestMarkTileComplete(t *testing.T) {
	rm := NewResumeManager("/tmp/tiles", "")
	rm.LoadResumeData()

	tile := model.Tile{X: 100, Y: 200, Z: 10}
	rm.MarkTileComplete(tile, "/tmp/tiles/10/100/200.png", 1024, "abc123")

	if len(rm.ResumeData.Completed) != 1 {
		t.Errorf("Expected 1 completed tile, got %d", len(rm.ResumeData.Completed))
	}

	info, ok := rm.ResumeData.Completed["10/100/200"]
	if !ok {
		t.Fatal("Tile not found in Completed map")
	}

	if info.X != 100 || info.Y != 200 || info.Z != 10 {
		t.Error("Tile coordinates not saved correctly")
	}

	if info.FileSize != 1024 {
		t.Errorf("Expected FileSize=1024, got %d", info.FileSize)
	}

	if info.MD5 != "abc123" {
		t.Errorf("Expected MD5=abc123, got %s", info.MD5)
	}
}

func TestMarkTileFailed(t *testing.T) {
	rm := NewResumeManager("/tmp/tiles", "")
	rm.LoadResumeData()

	tile := model.Tile{X: 100, Y: 200, Z: 10}
	rm.MarkTileFailed(tile, "HTTP 404")

	if len(rm.ResumeData.Failed) != 1 {
		t.Errorf("Expected 1 failed tile, got %d", len(rm.ResumeData.Failed))
	}

	reason, ok := rm.ResumeData.Failed["10/100/200"]
	if !ok {
		t.Fatal("Tile not found in Failed map")
	}

	if reason != "HTTP 404" {
		t.Errorf("Expected reason='HTTP 404', got %s", reason)
	}
}

func TestIsTileDownloaded(t *testing.T) {
	tempDir := t.TempDir()
	rm := NewResumeManager(tempDir, "")
	rm.LoadResumeData()

	tile := model.Tile{X: 100, Y: 200, Z: 10}

	downloaded, _ := rm.IsTileDownloaded(tile)
	if downloaded {
		t.Error("Tile should not be downloaded yet")
	}

	tilePath := filepath.Join(tempDir, "10", "100", "200.png")
	os.MkdirAll(filepath.Dir(tilePath), 0755)
	os.WriteFile(tilePath, make([]byte, 1024), 0644)

	rm.MarkTileComplete(tile, tilePath, 1024, "")

	downloaded, info := rm.IsTileDownloaded(tile)
	if !downloaded {
		t.Error("Tile should be downloaded")
	}

	if info == nil {
		t.Fatal("TileInfo should not be nil")
	}

	if info.FileSize != 1024 {
		t.Errorf("Expected FileSize=1024, got %d", info.FileSize)
	}
}

func TestIsTileDownloadedFileChanged(t *testing.T) {
	tempDir := t.TempDir()
	rm := NewResumeManager(tempDir, "")
	rm.LoadResumeData()

	tile := model.Tile{X: 100, Y: 200, Z: 10}
	tilePath := filepath.Join(tempDir, "10", "100", "200.png")
	os.MkdirAll(filepath.Dir(tilePath), 0755)
	os.WriteFile(tilePath, make([]byte, 1024), 0644)

	rm.MarkTileComplete(tile, tilePath, 1024, "")

	os.WriteFile(tilePath, make([]byte, 2048), 0644)

	downloaded, _ := rm.IsTileDownloaded(tile)
	if downloaded {
		t.Error("Tile should not be downloaded when file size changed")
	}

	rm.Mu.RLock()
	_, exists := rm.ResumeData.Completed["10/100/200"]
	rm.Mu.RUnlock()

	if exists {
		t.Error("Tile should be removed from Completed when file changed")
	}
}

func TestGetResumeData(t *testing.T) {
	rm := NewResumeManager("/tmp/tiles", "")
	rm.LoadResumeData()

	data := rm.GetResumeData()
	if data == nil {
		t.Fatal("GetResumeData returned nil")
	}

	if data != rm.ResumeData {
		t.Error("GetResumeData should return the same ResumeData")
	}
}

func TestConcurrentAccess(t *testing.T) {
	rm := NewResumeManager("/tmp/tiles", "")
	err := rm.LoadResumeData()
	if err != nil {
		t.Fatalf("LoadResumeData failed: %v", err)
	}

	if rm.ResumeData == nil {
		t.Fatal("ResumeData should not be nil after LoadResumeData")
	}

	var wg sync.WaitGroup
	const numGoroutines = 100
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			tile := model.Tile{X: i, Y: i, Z: 1}
			rm.MarkTileComplete(tile, "/tmp/tile.png", 1024, "")
		}(i)
	}

	wg.Wait()

	if len(rm.ResumeData.Completed) != numGoroutines {
		t.Errorf("Expected %d completed tiles, got %d", numGoroutines, len(rm.ResumeData.Completed))
	}
}

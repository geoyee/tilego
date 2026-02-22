package calculator

import (
	"testing"

	"github.com/geoyee/tilego/internal/model"
)

func TestNewTileCalculator(t *testing.T) {
	tc := NewTileCalculator()
	if tc == nil {
		t.Error("NewTileCalculator returned nil")
	}
}

func TestDeg2Num(t *testing.T) {
	tc := NewTileCalculator()

	tests := []struct {
		lon, lat float64
		zoom     int
		wantX    int
		wantY    int
	}{
		{0, 0, 0, 0, 0},
		{0, 0, 1, 1, 1},
		{-180, 85.0511, 1, 0, 0},
		{116.404, 39.915, 10, 843, 387},
	}

	for _, tt := range tests {
		x, y := tc.Deg2Num(tt.lon, tt.lat, tt.zoom)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("Deg2Num(%f, %f, %d) = (%d, %d), want (%d, %d)",
				tt.lon, tt.lat, tt.zoom, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestClampTileCoords(t *testing.T) {
	tc := NewTileCalculator()

	tests := []struct {
		minX, minY, maxX, maxY, zoom int
		wantMinX, wantMinY           int
		wantMaxX, wantMaxY           int
	}{
		{-1, -1, 2, 2, 1, 0, 0, 1, 1},
		{0, 0, 100, 100, 2, 0, 0, 3, 3},
		{5, 5, 10, 10, 3, 5, 5, 7, 7},
	}

	for _, tt := range tests {
		minX, minY, maxX, maxY := tc.ClampTileCoords(tt.minX, tt.minY, tt.maxX, tt.maxY, tt.zoom)
		if minX != tt.wantMinX || minY != tt.wantMinY || maxX != tt.wantMaxX || maxY != tt.wantMaxY {
			t.Errorf("ClampTileCoords(%d, %d, %d, %d, %d) = (%d, %d, %d, %d), want (%d, %d, %d, %d)",
				tt.minX, tt.minY, tt.maxX, tt.maxY, tt.zoom,
				minX, minY, maxX, maxY,
				tt.wantMinX, tt.wantMinY, tt.wantMaxX, tt.wantMaxY)
		}
	}
}

func TestCalculateTiles(t *testing.T) {
	tc := NewTileCalculator()

	tiles := tc.CalculateTiles(116.0, 39.0, 116.1, 39.1, 10, 10)

	if len(tiles) == 0 {
		t.Error("CalculateTiles returned empty slice")
	}

	for _, tile := range tiles {
		if tile.Z != 10 {
			t.Errorf("Expected zoom 10, got %d", tile.Z)
		}
	}
}

func TestCalculateTilesMultipleZooms(t *testing.T) {
	tc := NewTileCalculator()

	tiles := tc.CalculateTiles(116.0, 39.0, 116.1, 39.1, 8, 10)

	zoomCounts := make(map[int]int)
	for _, tile := range tiles {
		zoomCounts[tile.Z]++
	}

	if len(zoomCounts) != 3 {
		t.Errorf("Expected tiles from 3 zoom levels, got %d", len(zoomCounts))
	}

	for zoom := 8; zoom <= 10; zoom++ {
		if zoomCounts[zoom] == 0 {
			t.Errorf("No tiles for zoom level %d", zoom)
		}
	}
}

func TestCalculateTilesGlobal(t *testing.T) {
	tc := NewTileCalculator()

	tiles := tc.CalculateTiles(-180, -85, 180, 85, 0, 0)

	if len(tiles) != 1 {
		t.Errorf("Expected 1 tile at zoom 0, got %d", len(tiles))
	}

	if tiles[0].X != 0 || tiles[0].Y != 0 || tiles[0].Z != 0 {
		t.Errorf("Expected tile (0, 0, 0), got (%d, %d, %d)", tiles[0].X, tiles[0].Y, tiles[0].Z)
	}
}

func TestValidateZoomRange(t *testing.T) {
	tc := NewTileCalculator()

	tests := []struct {
		minZoom, maxZoom int
		wantErr          error
	}{
		{0, 18, nil},
		{5, 10, nil},
		{-1, 10, ErrInvalidZoomRange},
		{0, 19, ErrInvalidZoomRange},
		{10, 5, ErrInvalidZoomRange},
	}

	for _, tt := range tests {
		err := tc.ValidateZoomRange(tt.minZoom, tt.maxZoom)
		if err != tt.wantErr {
			t.Errorf("ValidateZoomRange(%d, %d) = %v, want %v",
				tt.minZoom, tt.maxZoom, err, tt.wantErr)
		}
	}
}

func TestValidateLatLonRange(t *testing.T) {
	tc := NewTileCalculator()

	tests := []struct {
		minLon, minLat, maxLon, maxLat float64
		wantErr                        bool
	}{
		{-180, -90, 180, 90, false},
		{116.0, 39.0, 117.0, 40.0, false},
		{-181, 0, 0, 0, true},
		{0, 0, 181, 0, true},
		{0, 0, 0, 0, true},
		{-180, -91, 180, 90, true},
		{-180, 0, 180, 91, true},
		{-180, 50, 180, 40, true},
	}

	for _, tt := range tests {
		err := tc.ValidateLatLonRange(tt.minLon, tt.minLat, tt.maxLon, tt.maxLat)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateLatLonRange(%f, %f, %f, %f) error = %v, wantErr %v",
				tt.minLon, tt.minLat, tt.maxLon, tt.maxLat, err, tt.wantErr)
		}
	}
}

func TestTileStructure(t *testing.T) {
	tile := model.Tile{X: 100, Y: 200, Z: 10}

	if tile.X != 100 || tile.Y != 200 || tile.Z != 10 {
		t.Errorf("Tile structure not correctly initialized")
	}
}

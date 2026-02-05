package calculator

import (
	"math"

	"github.com/geoyee/tilego/internal/model"
)

type TileCalculator struct{}

func NewTileCalculator() *TileCalculator {
	return &TileCalculator{}
}

func (tc *TileCalculator) CalculateTiles(minLon, minLat, maxLon, maxLat float64, minZoom, maxZoom int) []model.Tile {
	var totalTiles int
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		minX, minY := tc.Deg2Num(minLon, maxLat, zoom)
		maxX, maxY := tc.Deg2Num(maxLon, minLat, zoom)
		minX, minY, maxX, maxY = tc.ClampTileCoords(minX, minY, maxX, maxY, zoom)
		totalTiles += (maxX - minX + 1) * (maxY - minY + 1)
	}
	tiles := make([]model.Tile, 0, totalTiles)
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		minX, minY := tc.Deg2Num(minLon, maxLat, zoom)
		maxX, maxY := tc.Deg2Num(maxLon, minLat, zoom)
		minX, minY, maxX, maxY = tc.ClampTileCoords(minX, minY, maxX, maxY, zoom)
		for x := minX; x <= maxX; x++ {
			for y := minY; y <= maxY; y++ {
				tiles = append(tiles, model.Tile{
					X: x,
					Y: y,
					Z: zoom,
				})
			}
		}
	}
	return tiles
}

func (tc *TileCalculator) ClampTileCoords(minX, minY, maxX, maxY, zoom int) (int, int, int, int) {
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	maxTile := 1 << zoom
	if maxX >= maxTile {
		maxX = maxTile - 1
	}
	if maxY >= maxTile {
		maxY = maxTile - 1
	}
	return minX, minY, maxX, maxY
}

func (tc *TileCalculator) Deg2Num(lon, lat float64, zoom int) (x, y int) {
	tileCount := 1 << zoom
	n := float64(tileCount)
	x = int((lon + 180.0) / 360.0 * n)
	latRad := lat * math.Pi / 180.0
	y = int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)
	return x, y
}

func (tc *TileCalculator) ValidateZoomRange(minZoom, maxZoom int) error {
	if minZoom < 0 || maxZoom > 18 || minZoom > maxZoom {
		return ErrInvalidZoomRange
	}
	return nil
}

func (tc *TileCalculator) ValidateLatLonRange(minLon, minLat, maxLon, maxLat float64) error {
	if minLon < -180 || maxLon > 180 || minLon >= maxLon {
		return ErrInvalidLonRange
	}
	if minLat < -90 || maxLat > 90 || minLat >= maxLat {
		return ErrInvalidLatRange
	}
	return nil
}

// Package calculator provides tile coordinate calculation functionality.
//
// This package handles the conversion between geographic coordinates (latitude/longitude)
// and tile coordinates (x, y, z) for map tile downloads.
package calculator

import (
	"math"

	"github.com/geoyee/tilego/internal/model"
)

// TileCalculator handles tile coordinate calculations.
type TileCalculator struct{}

// NewTileCalculator creates a new TileCalculator instance.
func NewTileCalculator() *TileCalculator {
	return &TileCalculator{}
}

// CalculateTiles computes all tiles within the specified geographic bounds and zoom range.
//
// Parameters:
//   - minLon: Minimum longitude (-180 to 180)
//   - minLat: Minimum latitude (-90 to 90)
//   - maxLon: Maximum longitude (-180 to 180)
//   - maxLat: Maximum latitude (-90 to 90)
//   - minZoom: Minimum zoom level (0-18)
//   - maxZoom: Maximum zoom level (0-18)
//
// Returns a slice of Tile coordinates within the specified range.
func (tc *TileCalculator) CalculateTiles(minLon, minLat, maxLon, maxLat float64, minZoom, maxZoom int) ([]model.Tile, error) {
	var totalTiles int
	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		minX, minY := tc.Deg2Num(minLon, maxLat, zoom)
		maxX, maxY := tc.Deg2Num(maxLon, minLat, zoom)

		minX, minY, maxX, maxY = tc.clampTileCoords(minX, minY, maxX, maxY, zoom)
		totalTiles += (maxX - minX + 1) * (maxY - minY + 1)
	}

	tiles := make([]model.Tile, 0, totalTiles)

	for zoom := minZoom; zoom <= maxZoom; zoom++ {
		minX, minY := tc.Deg2Num(minLon, maxLat, zoom)
		maxX, maxY := tc.Deg2Num(maxLon, minLat, zoom)

		minX, minY, maxX, maxY = tc.clampTileCoords(minX, minY, maxX, maxY, zoom)

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

	return tiles, nil
}

// clampTileCoords ensures tile coordinates are within valid bounds for the given zoom level.
func (tc *TileCalculator) clampTileCoords(minX, minY, maxX, maxY, zoom int) (int, int, int, int) {
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

// Deg2Num converts geographic coordinates to tile coordinates.
//
// Parameters:
//   - lon: Longitude in degrees (-180 to 180)
//   - lat: Latitude in degrees (-90 to 90)
//   - zoom: Zoom level (0-18)
//
// Returns the tile x and y coordinates.
func (tc *TileCalculator) Deg2Num(lon, lat float64, zoom int) (x, y int) {
	tileCount := 1 << zoom
	n := float64(tileCount)
	x = int((lon + 180.0) / 360.0 * n)

	latRad := lat * math.Pi / 180.0
	y = int((1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n)

	return x, y
}

// ValidateZoomRange validates that the zoom level range is valid.
func (tc *TileCalculator) ValidateZoomRange(minZoom, maxZoom int) error {
	if minZoom < 0 || maxZoom > 18 {
		return ErrInvalidZoomRange
	}
	if minZoom > maxZoom {
		return ErrInvalidZoomRange
	}
	return nil
}

// ValidateLatLonRange validates that the geographic coordinate bounds are valid.
func (tc *TileCalculator) ValidateLatLonRange(minLon, minLat, maxLon, maxLat float64) error {
	if minLon < -180 || maxLon > 180 {
		return ErrInvalidLonRange
	}
	if minLat < -90 || maxLat > 90 {
		return ErrInvalidLatRange
	}
	if minLon >= maxLon {
		return ErrInvalidLonRange
	}
	if minLat >= maxLat {
		return ErrInvalidLatRange
	}
	return nil
}

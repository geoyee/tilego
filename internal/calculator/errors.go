// Package calculator provides tile coordinate calculation functionality.
//
// This package handles the conversion between geographic coordinates (latitude/longitude)
// and tile coordinates (x, y, z) for map tile downloads.
package calculator

import "errors"

// Error definitions for tile coordinate calculations.
var (
	// ErrInvalidZoomRange indicates that the zoom level range is invalid.
	// Valid range is 0 <= minZoom <= maxZoom <= 18.
	ErrInvalidZoomRange = errors.New("invalid zoom level range (0 <= min-zoom <= max-zoom <= 18)")

	// ErrInvalidLonRange indicates that the longitude range is invalid.
	// Valid range is -180 <= min-lon < max-lon <= 180.
	ErrInvalidLonRange = errors.New("invalid longitude range (-180 <= min-lon < max-lon <= 180)")

	// ErrInvalidLatRange indicates that the latitude range is invalid.
	// Valid range is -90 <= min-lat < max-lat <= 90.
	ErrInvalidLatRange = errors.New("invalid latitude range (-90 <= min-lat < max-lat <= 90)")

	// ErrNoTilesFound indicates that no tiles were found within the specified range.
	ErrNoTilesFound = errors.New("no tiles found in the specified range")
)

package calculator

import "errors"

var (
	ErrInvalidZoomRange = errors.New("invalid zoom level range (0 <= min-zoom <= max-zoom <= 18)")
	ErrInvalidLonRange  = errors.New("invalid longitude range (-180 <= min-lon < max-lon <= 180)")
	ErrInvalidLatRange  = errors.New("invalid latitude range (-90 <= min-lat < max-lat <= 90)")
	ErrNoTilesFound     = errors.New("no tiles found in the specified range")
)

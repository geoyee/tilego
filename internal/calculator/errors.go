// Package calculator 提供瓦片坐标计算功能
package calculator

import "errors"

var (
	// ErrInvalidZoomRange 无效的缩放级别范围
	ErrInvalidZoomRange = errors.New("缩放级别无效 (0 <= min-zoom <= max-zoom <= 18)")
	// ErrInvalidLonRange 无效的经度范围
	ErrInvalidLonRange = errors.New("经度范围无效 (-180 <= min-lon < max-lon <= 180)")
	// ErrInvalidLatRange 无效的纬度范围
	ErrInvalidLatRange = errors.New("纬度范围无效 (-90 <= min-lat < max-lat <= 90)")
	// ErrNoTilesFound 没有找到范围内的瓦片
	ErrNoTilesFound = errors.New("没有找到范围内的瓦片")
)

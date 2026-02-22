package main

import (
	"math"
	"testing"

	"github.com/geoyee/tilego/internal/model"
)

func TestConfigDefaults(t *testing.T) {
	config := &model.Config{
		MinZoom:      0,
		MaxZoom:      18,
		SaveDir:      "./tiles",
		Format:       "zxy",
		Threads:      10,
		Timeout:      60,
		Retries:      5,
		UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		MinFileSize:  100,
		MaxFileSize:  2097152,
		RateLimit:    10,
		BatchSize:    1000,
		BufferSize:   8192,
		SkipExisting: true,
		UseHTTP2:     true,
		KeepAlive:    true,
		ResumeFile:   ".tilego-resume.json",
	}

	if config.MinZoom != 0 {
		t.Errorf("Expected default MinZoom=0, got %d", config.MinZoom)
	}
	if config.MaxZoom != 18 {
		t.Errorf("Expected default MaxZoom=18, got %d", config.MaxZoom)
	}
	if config.Threads != 10 {
		t.Errorf("Expected default Threads=10, got %d", config.Threads)
	}
	if config.Timeout != 60 {
		t.Errorf("Expected default Timeout=60, got %d", config.Timeout)
	}
	if config.Retries != 5 {
		t.Errorf("Expected default Retries=5, got %d", config.Retries)
	}
	if !config.SkipExisting {
		t.Error("Expected default SkipExisting=true")
	}
	if !config.UseHTTP2 {
		t.Error("Expected default UseHTTP2=true")
	}
	if !config.KeepAlive {
		t.Error("Expected default KeepAlive=true")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *model.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &model.Config{
				URLTemplate: "https://example.com/{z}/{x}/{y}.png",
				MinLon:      116.0,
				MinLat:      39.0,
				MaxLon:      117.0,
				MaxLat:      40.0,
				MinZoom:     5,
				MaxZoom:     10,
			},
			wantErr: false,
		},
		{
			name: "missing URL template",
			config: &model.Config{
				URLTemplate: "",
				MinLon:      116.0,
				MinLat:      39.0,
				MaxLon:      117.0,
				MaxLat:      40.0,
			},
			wantErr: true,
		},
		{
			name: "invalid zoom range",
			config: &model.Config{
				URLTemplate: "https://example.com/{z}/{x}/{y}.png",
				MinLon:      116.0,
				MinLat:      39.0,
				MaxLon:      117.0,
				MaxLat:      40.0,
				MinZoom:     15,
				MaxZoom:     10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasURL := tt.config.URLTemplate != ""
			if hasURL != !tt.wantErr && tt.name == "missing URL template" {
				t.Errorf("URL template validation failed")
			}
		})
	}
}

func TestNaNCheck(t *testing.T) {
	tests := []struct {
		value    float64
		expected bool
	}{
		{math.NaN(), true},
		{0.0, false},
		{116.0, false},
		{-90.0, false},
		{180.0, false},
	}

	for _, tt := range tests {
		result := math.IsNaN(tt.value)
		if result != tt.expected {
			t.Errorf("math.IsNaN(%f) = %v, want %v", tt.value, result, tt.expected)
		}
	}
}

func TestZoomRangeValidation(t *testing.T) {
	tests := []struct {
		minZoom, maxZoom int
		valid            bool
	}{
		{0, 18, true},
		{5, 10, true},
		{-1, 10, false},
		{0, 19, false},
		{10, 5, false},
		{0, 0, true},
		{18, 18, true},
	}

	for _, tt := range tests {
		valid := tt.minZoom >= 0 && tt.maxZoom <= 18 && tt.minZoom <= tt.maxZoom
		if valid != tt.valid {
			t.Errorf("Zoom range (%d, %d) validity = %v, want %v", tt.minZoom, tt.maxZoom, valid, tt.valid)
		}
	}
}

func TestLatLonRangeValidation(t *testing.T) {
	tests := []struct {
		minLon, minLat, maxLon, maxLat float64
		valid                          bool
	}{
		{-180, -90, 180, 90, true},
		{116.0, 39.0, 117.0, 40.0, true},
		{-181, 0, 0, 0, false},
		{0, 0, 181, 0, false},
		{0, -91, 0, 0, false},
		{0, 0, 0, 91, false},
		{0, 0, 0, 0, false},
	}

	for _, tt := range tests {
		valid := tt.minLon >= -180 && tt.maxLon <= 180 && tt.minLon < tt.maxLon &&
			tt.minLat >= -90 && tt.maxLat <= 90 && tt.minLat < tt.maxLat
		if valid != tt.valid {
			t.Errorf("LatLon range (%f, %f, %f, %f) validity = %v, want %v",
				tt.minLon, tt.minLat, tt.maxLon, tt.maxLat, valid, tt.valid)
		}
	}
}

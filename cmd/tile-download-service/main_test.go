package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleHealth(t *testing.T) {
	server := NewServer(8765, []string{"*"})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

func TestHandleDownload(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	reqBody := DownloadRequest{
		URLTemplate: "https://tile.example.com/{z}/{x}/{y}.png",
		MinLon:      116.0,
		MinLat:      39.0,
		MaxLon:      116.1,
		MaxLat:      39.1,
		MinZoom:     10,
		MaxZoom:     10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleDownload(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected Data to be a map")
	}

	if _, ok := data["task_id"]; !ok {
		t.Error("Expected task_id in response")
	}
}

func TestHandleDownloadWithID(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	reqBody := DownloadRequest{
		ID:          "test_task_123",
		URLTemplate: "https://tile.example.com/{z}/{x}/{y}.png",
		MinLon:      116.0,
		MinLat:      39.0,
		MaxLon:      116.1,
		MaxLat:      39.1,
		MinZoom:     10,
		MaxZoom:     10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleDownload(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected Data to be a map")
	}

	if data["task_id"] != "test_task_123" {
		t.Errorf("Expected task_id to be test_task_123, got %v", data["task_id"])
	}
}

func TestHandleDownloadMissingURLTemplate(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	reqBody := DownloadRequest{
		MinLon: 116.0,
		MinLat: 39.0,
		MaxLon: 116.1,
		MaxLat: 39.1,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleDownload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleStatus(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	task := server.taskManager.CreateTask("test_status_task", defaultConfig())
	task.mu.Lock()
	task.Status = StatusPending
	task.Total = 100
	task.Success = 50
	task.Failed = 5
	task.Skipped = 10
	task.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/status/test_status_task", nil)
	w := httptest.NewRecorder()

	server.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

func TestHandleStatusNotFound(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	req := httptest.NewRequest(http.MethodGet, "/api/status/nonexistent_task", nil)
	w := httptest.NewRecorder()

	server.handleStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleTasks(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	server.taskManager.CreateTask("task1", defaultConfig())
	server.taskManager.CreateTask("task2", defaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	w := httptest.NewRecorder()

	server.handleTasks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	tasks, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("Expected Data to be an array")
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestHandleStop(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	task := server.taskManager.CreateTask("test_stop_task", defaultConfig())
	task.mu.Lock()
	task.Status = StatusRunning
	task.done = make(chan struct{})
	task.mu.Unlock()

	go func() {
		time.Sleep(100 * time.Millisecond)
		task.mu.Lock()
		task.Status = StatusStopped
		close(task.done)
		task.mu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodPost, "/api/stop/test_stop_task", nil)
	w := httptest.NewRecorder()

	server.handleStop(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

func TestHandleStopNotRunning(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	task := server.taskManager.CreateTask("test_stop_not_running", defaultConfig())
	task.mu.Lock()
	task.Status = StatusPending
	task.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/stop/test_stop_not_running", nil)
	w := httptest.NewRecorder()

	server.handleStop(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleStopNotFound(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	req := httptest.NewRequest(http.MethodPost, "/api/stop/nonexistent_task", nil)
	w := httptest.NewRecorder()

	server.handleStop(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleDelete(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	server.taskManager.CreateTask("test_delete_task", defaultConfig())

	req := httptest.NewRequest(http.MethodDelete, "/api/delete/test_delete_task", nil)
	w := httptest.NewRecorder()

	server.handleDelete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}

	if _, ok := server.taskManager.GetTask("test_delete_task"); ok {
		t.Error("Task should be deleted")
	}
}

func TestHandleDeleteNotFound(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	req := httptest.NewRequest(http.MethodDelete, "/api/delete/nonexistent_task", nil)
	w := httptest.NewRecorder()

	server.handleDelete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleDeleteRunningTask(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	task := server.taskManager.CreateTask("test_delete_running", defaultConfig())
	task.mu.Lock()
	task.Status = StatusRunning
	task.done = make(chan struct{})
	task.mu.Unlock()

	go func() {
		time.Sleep(100 * time.Millisecond)
		task.mu.Lock()
		close(task.done)
		task.mu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodDelete, "/api/delete/test_delete_running", nil)
	w := httptest.NewRecorder()

	server.handleDelete(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if _, ok := server.taskManager.GetTask("test_delete_running"); ok {
		t.Error("Task should be deleted")
	}
}

func TestCORS(t *testing.T) {
	server := NewServer(8765, []string{"http://localhost:3000", "http://example.com"})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", server.handleHealth)
	handler := server.corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Unexpected CORS origin: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSWildcard(t *testing.T) {
	server := NewServer(8765, []string{"*"})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", server.handleHealth)
	handler := server.corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected wildcard CORS origin, got: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestDownloadRequestToConfig(t *testing.T) {
	skipExisting := false
	checkMD5 := true
	useHTTP2 := false
	keepAlive := false

	req := &DownloadRequest{
		URLTemplate:  "https://tile.example.com/{z}/{x}/{y}.png",
		MinLon:       116.0,
		MinLat:       39.0,
		MaxLon:       117.0,
		MaxLat:       40.0,
		MinZoom:      5,
		MaxZoom:      15,
		SaveDir:      "/tmp/tiles",
		Format:       "xyz",
		Threads:      20,
		Timeout:      120,
		Retries:      3,
		ProxyURL:     "http://proxy:8080",
		UserAgent:    "TestAgent",
		Referer:      "http://referer.com",
		SkipExisting: &skipExisting,
		CheckMD5:     &checkMD5,
		MinFileSize:  500,
		MaxFileSize:  5000000,
		RateLimit:    50,
		UseHTTP2:     &useHTTP2,
		KeepAlive:    &keepAlive,
		BatchSize:    500,
		BufferSize:   16384,
	}

	config := req.ToConfig()

	if config.URLTemplate != "https://tile.example.com/{z}/{x}/{y}.png" {
		t.Errorf("Unexpected URLTemplate: %s", config.URLTemplate)
	}
	if config.MinZoom != 5 {
		t.Errorf("Expected MinZoom 5, got %d", config.MinZoom)
	}
	if config.MaxZoom != 15 {
		t.Errorf("Expected MaxZoom 15, got %d", config.MaxZoom)
	}
	if config.Threads != 20 {
		t.Errorf("Expected Threads 20, got %d", config.Threads)
	}
	if config.SkipExisting != false {
		t.Errorf("Expected SkipExisting false, got %v", config.SkipExisting)
	}
	if config.CheckMD5 != true {
		t.Errorf("Expected CheckMD5 true, got %v", config.CheckMD5)
	}
	if config.UseHTTP2 != false {
		t.Errorf("Expected UseHTTP2 false, got %v", config.UseHTTP2)
	}
	if config.KeepAlive != false {
		t.Errorf("Expected KeepAlive false, got %v", config.KeepAlive)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := defaultConfig()

	if config.MinZoom != 0 {
		t.Errorf("Expected default MinZoom 0, got %d", config.MinZoom)
	}
	if config.MaxZoom != 18 {
		t.Errorf("Expected default MaxZoom 18, got %d", config.MaxZoom)
	}
	if config.Threads != 10 {
		t.Errorf("Expected default Threads 10, got %d", config.Threads)
	}
	if config.SkipExisting != true {
		t.Errorf("Expected default SkipExisting true, got %v", config.SkipExisting)
	}
	if config.UseHTTP2 != true {
		t.Errorf("Expected default UseHTTP2 true, got %v", config.UseHTTP2)
	}
	if config.KeepAlive != true {
		t.Errorf("Expected default KeepAlive true, got %v", config.KeepAlive)
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/geoyee/tilego/internal/download"
	"github.com/geoyee/tilego/internal/model"
)

type TaskStatus string

const (
	StatusPending         TaskStatus = "pending"
	StatusRunning         TaskStatus = "running"
	StatusStopped         TaskStatus = "stopped"
	StatusComplete        TaskStatus = "complete"
	StatusCompletePartial TaskStatus = "complete_partial"
	StatusFailed          TaskStatus = "failed"
)

type Task struct {
	ID          string            `json:"id"`
	Config      *model.Config     `json:"config"`
	Status      TaskStatus        `json:"status"`
	Progress    float64           `json:"progress"`
	Total       int64             `json:"total"`
	Success     int64             `json:"success"`
	Failed      int64             `json:"failed"`
	Skipped     int64             `json:"skipped"`
	BytesTotal  int64             `json:"bytes_total"`
	Speed       float64           `json:"speed"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time,omitempty"`
	Error       string            `json:"error,omitempty"`
	cancelFunc  context.CancelFunc `json:"-"`
	downloader  *download.Downloader `json:"-"`
	mu          sync.RWMutex       `json:"-"`
}

type TaskManager struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
	}
}

func (tm *TaskManager) CreateTask(id string, config *model.Config) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task := &Task{
		ID:     id,
		Config: config,
		Status: StatusPending,
	}
	tm.tasks[id] = task
	return task
}

func (tm *TaskManager) GetTask(id string) (*Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	task, ok := tm.tasks[id]
	return task, ok
}

func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (tm *TaskManager) DeleteTask(id string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if task, ok := tm.tasks[id]; ok {
		if task.Status == StatusRunning && task.cancelFunc != nil {
			task.cancelFunc()
		}
		delete(tm.tasks, id)
		return true
	}
	return false
}

type DownloadRequest struct {
	ID          string  `json:"id,omitempty"`
	URLTemplate string  `json:"url_template"`
	MinLon      float64 `json:"min_lon"`
	MinLat      float64 `json:"min_lat"`
	MaxLon      float64 `json:"max_lon"`
	MaxLat      float64 `json:"max_lat"`
	MinZoom     int     `json:"min_zoom,omitempty"`
	MaxZoom     int     `json:"max_zoom,omitempty"`
	SaveDir     string  `json:"save_dir,omitempty"`
	Format      string  `json:"format,omitempty"`
	Threads     int     `json:"threads,omitempty"`
	Timeout     int     `json:"timeout,omitempty"`
	Retries     int     `json:"retries,omitempty"`
	ProxyURL    string  `json:"proxy_url,omitempty"`
	UserAgent   string  `json:"user_agent,omitempty"`
	Referer     string  `json:"referer,omitempty"`
	SkipExisting bool   `json:"skip_existing,omitempty"`
	CheckMD5    bool    `json:"check_md5,omitempty"`
	MinFileSize int64   `json:"min_file_size,omitempty"`
	MaxFileSize int64   `json:"max_file_size,omitempty"`
	RateLimit   int     `json:"rate_limit,omitempty"`
	UseHTTP2    bool    `json:"use_http2,omitempty"`
	KeepAlive   bool    `json:"keep_alive,omitempty"`
	BatchSize   int     `json:"batch_size,omitempty"`
	BufferSize  int     `json:"buffer_size,omitempty"`
}

func defaultConfig() *model.Config {
	return &model.Config{
		MinZoom:     0,
		MaxZoom:     18,
		SaveDir:     "./tiles",
		Format:      "zxy",
		Threads:     10,
		Timeout:     60,
		Retries:     5,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		MinFileSize: 100,
		MaxFileSize: 2097152,
		RateLimit:   10,
		BatchSize:   1000,
		BufferSize:  8192,
	}
}

func (r *DownloadRequest) ToConfig() *model.Config {
	config := defaultConfig()
	config.URLTemplate = r.URLTemplate
	config.MinLon = r.MinLon
	config.MinLat = r.MinLat
	config.MaxLon = r.MaxLon
	config.MaxLat = r.MaxLat
	if r.MinZoom != 0 {
		config.MinZoom = r.MinZoom
	}
	if r.MaxZoom != 0 {
		config.MaxZoom = r.MaxZoom
	}
	if r.SaveDir != "" {
		config.SaveDir = r.SaveDir
	}
	if r.Format != "" {
		config.Format = r.Format
	}
	if r.Threads != 0 {
		config.Threads = r.Threads
	}
	if r.Timeout != 0 {
		config.Timeout = r.Timeout
	}
	if r.Retries != 0 {
		config.Retries = r.Retries
	}
	if r.ProxyURL != "" {
		config.ProxyURL = r.ProxyURL
	}
	if r.UserAgent != "" {
		config.UserAgent = r.UserAgent
	}
	if r.Referer != "" {
		config.Referer = r.Referer
	}
	config.SkipExisting = r.SkipExisting
	config.CheckMD5 = r.CheckMD5
	if r.MinFileSize != 0 {
		config.MinFileSize = r.MinFileSize
	}
	if r.MaxFileSize != 0 {
		config.MaxFileSize = r.MaxFileSize
	}
	if r.RateLimit != 0 {
		config.RateLimit = r.RateLimit
	}
	config.UseHTTP2 = r.UseHTTP2
	config.KeepAlive = r.KeepAlive
	if r.BatchSize != 0 {
		config.BatchSize = r.BatchSize
	}
	if r.BufferSize != 0 {
		config.BufferSize = r.BufferSize
	}
	return config
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type Server struct {
	taskManager  *TaskManager
	port         int
	allowedOrigins []string
}

func NewServer(port int, allowedOrigins []string) *Server {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	return &Server{
		taskManager:    NewTaskManager(),
		port:           port,
		allowedOrigins: allowedOrigins,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/download", s.handleDownload)
	mux.HandleFunc("/api/status/", s.handleStatus)
	mux.HandleFunc("/api/stop/", s.handleStop)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/delete/", s.handleDelete)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Tile Download Service starting on %s", addr)
	log.Println("API Endpoints:")
	log.Println("  POST /api/download  - Start a new download task")
	log.Println("  GET  /api/status/{id} - Get task status")
	log.Println("  POST /api/stop/{id}   - Stop a running task")
	log.Println("  GET  /api/tasks       - List all tasks")
	log.Println("  DELETE /api/delete/{id} - Delete a task")
	log.Println("  GET  /api/health      - Health check")

	return http.ListenAndServe(addr, s.corsMiddleware(mux))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, allowedOrigin := range s.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				if allowedOrigin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				break
			}
		}
		if !allowed && len(s.allowedOrigins) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", s.allowedOrigins[0])
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"status": "healthy", "time": time.Now().Format(time.RFC3339)},
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.URLTemplate == "" {
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "url_template is required",
		})
		return
	}
	if math.IsNaN(req.MinLon) || math.IsNaN(req.MaxLon) || math.IsNaN(req.MinLat) || math.IsNaN(req.MaxLat) {
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "min_lon, max_lon, min_lat, max_lat are required",
		})
		return
	}

	taskID := req.ID
	if taskID == "" {
		taskID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	config := req.ToConfig()
	task := s.taskManager.CreateTask(taskID, config)

	go s.runDownloadTask(task)

	s.respondJSON(w, http.StatusAccepted, APIResponse{
		Success: true,
		Message: "Download task created",
		Data:    map[string]string{"task_id": taskID},
	})
}

func (s *Server) runDownloadTask(task *Task) {
	ctx, cancel := context.WithCancel(context.Background())
	task.mu.Lock()
	task.Status = StatusRunning
	task.StartTime = time.Now()
	task.cancelFunc = cancel
	task.mu.Unlock()

	defer func() {
		task.mu.Lock()
		task.EndTime = time.Now()
		if task.Status == StatusRunning {
			if task.Success == 0 && task.Failed > 0 {
				task.Status = StatusFailed
			} else if task.Failed > 0 {
				task.Status = StatusCompletePartial
			} else {
				task.Status = StatusComplete
			}
		}
		task.mu.Unlock()
	}()

	config := task.Config
	downloader := download.NewDownloader(config)

	task.mu.Lock()
	task.downloader = downloader
	task.mu.Unlock()

	if err := downloader.Init(); err != nil {
		task.mu.Lock()
		task.Status = StatusFailed
		task.Error = fmt.Sprintf("Initialization failed: %v", err)
		task.mu.Unlock()
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if downloader.StatsMonitor != nil {
					stats := downloader.StatsMonitor.GetStats()
					task.mu.Lock()
					task.Total = stats.Total
					task.Success = stats.Success
					task.Failed = stats.Failed
					task.Skipped = stats.Skipped
					task.BytesTotal = stats.BytesTotal
					if stats.Total > 0 {
						task.Progress = float64(stats.Success+stats.Skipped) / float64(stats.Total) * 100
					}
					if len(stats.SpeedHistory) > 0 {
						task.Speed = stats.SpeedHistory[len(stats.SpeedHistory)-1].Speed
					}
					task.mu.Unlock()
				}
			}
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- downloader.Run()
	}()

	select {
	case <-ctx.Done():
		downloader.Cleanup()
		task.mu.Lock()
		task.Status = StatusStopped
		task.mu.Unlock()
	case err := <-done:
		if err != nil {
			task.mu.Lock()
			task.Error = err.Error()
			task.mu.Unlock()
		}
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	taskID := strings.TrimPrefix(r.URL.Path, "/api/status/")
	if taskID == "" || taskID == r.URL.Path {
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Task ID is required",
		})
		return
	}

	task, ok := s.taskManager.GetTask(taskID)
	if !ok {
		s.respondJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Message: "Task not found",
		})
		return
	}

	task.mu.RLock()
	status := map[string]interface{}{
		"id":          task.ID,
		"status":      task.Status,
		"progress":    task.Progress,
		"total":       task.Total,
		"success":     task.Success,
		"failed":      task.Failed,
		"skipped":     task.Skipped,
		"bytes_total": task.BytesTotal,
		"speed":       task.Speed,
		"start_time":  task.StartTime,
	}
	if !task.EndTime.IsZero() {
		status["end_time"] = task.EndTime
		status["duration"] = task.EndTime.Sub(task.StartTime).String()
	}
	if task.Error != "" {
		status["error"] = task.Error
	}
	task.mu.RUnlock()

	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    status,
	})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	taskID := strings.TrimPrefix(r.URL.Path, "/api/stop/")
	if taskID == "" || taskID == r.URL.Path {
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Task ID is required",
		})
		return
	}

	task, ok := s.taskManager.GetTask(taskID)
	if !ok {
		s.respondJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Message: "Task not found",
		})
		return
	}

	task.mu.Lock()
	if task.Status != StatusRunning {
		task.mu.Unlock()
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: fmt.Sprintf("Task is not running (current status: %s)", task.Status),
		})
		return
	}

	if task.cancelFunc != nil {
		task.cancelFunc()
	}
	task.Status = StatusStopped
	task.mu.Unlock()

	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Task stopped",
	})
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	tasks := s.taskManager.ListTasks()
	result := make([]map[string]interface{}, 0, len(tasks))
	for _, task := range tasks {
		task.mu.RLock()
		t := map[string]interface{}{
			"id":          task.ID,
			"status":      task.Status,
			"progress":    task.Progress,
			"total":       task.Total,
			"success":     task.Success,
			"failed":      task.Failed,
			"skipped":     task.Skipped,
			"bytes_total": task.BytesTotal,
			"speed":       task.Speed,
			"start_time":  task.StartTime,
		}
		if !task.EndTime.IsZero() {
			t["end_time"] = task.EndTime
		}
		if task.Error != "" {
			t["error"] = task.Error
		}
		task.mu.RUnlock()
		result = append(result, t)
	}

	s.respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.respondJSON(w, http.StatusMethodNotAllowed, APIResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	taskID := strings.TrimPrefix(r.URL.Path, "/api/delete/")
	if taskID == "" || taskID == r.URL.Path {
		s.respondJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Task ID is required",
		})
		return
	}

	if s.taskManager.DeleteTask(taskID) {
		s.respondJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Task deleted",
		})
	} else {
		s.respondJSON(w, http.StatusNotFound, APIResponse{
			Success: false,
			Message: "Task not found",
		})
	}
}

func main() {
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := fmt.Sscanf(portStr, "%d", &port); err != nil || p != 1 {
			log.Printf("Invalid PORT environment variable, using default: %d", port)
		}
	}

	allowedOrigins := []string{"*"}
	if originsStr := os.Getenv("ALLOWED_ORIGINS"); originsStr != "" {
		allowedOrigins = strings.Split(originsStr, ",")
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	}

	log.Printf("Starting Tile Download Service...")
	log.Printf("Allowed CORS origins: %v", allowedOrigins)

	server := NewServer(port, allowedOrigins)
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

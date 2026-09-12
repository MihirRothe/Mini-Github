package health

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"forgehub/apps/api/internal/database"
	"forgehub/apps/api/internal/redis"
)

var startTime = time.Now()

type Handler struct {
	db    *database.DB
	redis *redis.Client
}

func NewHandler(db *database.DB, redis *redis.Client) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

type TelemetryResponse struct {
	Status        string              `json:"status"`
	Version       string              `json:"version"`
	UptimeSeconds int64               `json:"uptime_seconds"`
	Timestamp     time.Time           `json:"timestamp"`
	Database      database.HealthInfo `json:"database"`
	Redis         redis.HealthInfo    `json:"redis"`
	System        SystemMetrics       `json:"system"`
}

type SystemMetrics struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	AllocMB      uint64 `json:"alloc_mb"`
	TotalAllocMB uint64 `json:"total_alloc_mb"`
	SysMB        uint64 `json:"sys_mb"`
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var dbHealth database.HealthInfo
	if h.db != nil {
		dbHealth = h.db.Health(ctx)
	} else {
		dbHealth = database.HealthInfo{Status: "unconfigured"}
	}

	var redisHealth redis.HealthInfo
	if h.redis != nil {
		redisHealth = h.redis.Health(ctx)
	} else {
		redisHealth = redis.HealthInfo{Status: "unconfigured"}
	}

	overallStatus := "healthy"
	if dbHealth.Status == "down" || redisHealth.Status == "down" {
		overallStatus = "degraded"
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	res := TelemetryResponse{
		Status:        overallStatus,
		Version:       "0.1.0-alpha",
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		Timestamp:     time.Now().UTC(),
		Database:      dbHealth,
		Redis:         redisHealth,
		System: SystemMetrics{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			AllocMB:      mem.Alloc / 1024 / 1024,
			TotalAllocMB: mem.TotalAlloc / 1024 / 1024,
			SysMB:        mem.Sys / 1024 / 1024,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if h.db != nil && !h.db.IsStandalone() {
		if err := h.db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Database unavailable"))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("READY"))
}

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonny/aegis/v14/internal/api/response"
	"github.com/wonny/aegis/v14/internal/infra/database/postgres"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	dbPool    *postgres.Pool
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(dbPool *postgres.Pool, version string) *HealthHandler {
	return &HealthHandler{
		dbPool:    dbPool,
		startTime: time.Now(),
		version:   version,
	}
}

// SimpleHealthResponse represents a simple health check response
type SimpleHealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadyResponse represents a readiness check response
type ReadyResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message,omitempty"`
}

// DetailedHealthResponse represents detailed health information
type DetailedHealthResponse struct {
	Status         string                        `json:"status"`
	Version        string                        `json:"version"`
	UptimeSeconds  int64                         `json:"uptime_seconds"`
	Timestamp      time.Time                     `json:"timestamp"`
	Components     map[string]ComponentHealth    `json:"components"`
}

// ComponentHealth represents health status of a component
type ComponentHealth struct {
	Status       string                 `json:"status"`
	ResponseTime string                 `json:"response_time"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Message      string                 `json:"message,omitempty"`
}

// Health returns simple liveness check
// GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	response := SimpleHealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusOK, response)
}

// Ready returns readiness check with dependency checks
// GET /health/ready
func (h *HealthHandler) Ready(c *gin.Context) {
	checks := make(map[string]string)
	allReady := true
	message := ""

	// Check database
	dbHealth := h.dbPool.Health(c.Request.Context())
	if dbHealth.Status == "healthy" {
		checks["database"] = "ok"
	} else {
		checks["database"] = "error"
		allReady = false
		message = "Database connection failed"
	}

	// TODO: Check Redis when implemented
	// redisHealth := h.redis.Ping(c.Request.Context())
	// if redisHealth.Err() == nil {
	// 	checks["redis"] = "ok"
	// } else {
	// 	checks["redis"] = "error"
	// 	allReady = false
	// 	if message == "" {
	// 		message = "Redis connection failed"
	// 	}
	// }

	status := "ready"
	statusCode := http.StatusOK

	if !allReady {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	resp := ReadyResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
		Message:   message,
	}

	c.JSON(statusCode, resp)
}

// Detailed returns detailed system health information
// GET /api/health/detailed
func (h *HealthHandler) Detailed(c *gin.Context) {
	components := make(map[string]ComponentHealth)
	overallStatus := "healthy"

	// Check database
	dbHealth := h.dbPool.Health(c.Request.Context())

	dbComponent := ComponentHealth{
		Status:       dbHealth.Status,
		ResponseTime: dbHealth.ResponseTime,
		Details: map[string]interface{}{
			"active_conns": dbHealth.ActiveConns,
			"idle_conns":   dbHealth.IdleConns,
			"total_conns":  dbHealth.TotalConns,
			"max_conns":    dbHealth.MaxConns,
		},
	}

	if dbHealth.Error != "" {
		dbComponent.Message = dbHealth.Error
	}

	components["database"] = dbComponent

	// Determine overall status
	if dbHealth.Status == "unhealthy" {
		overallStatus = "unhealthy"
	} else if dbHealth.Status == "degraded" && overallStatus != "unhealthy" {
		overallStatus = "degraded"
	}

	// TODO: Add Redis check
	// redisHealth := h.checkRedis(c.Request.Context())
	// components["redis"] = redisHealth

	detailedResponse := DetailedHealthResponse{
		Status:        overallStatus,
		Version:       h.version,
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
		Timestamp:     time.Now(),
		Components:    components,
	}

	response.Success(c, detailedResponse)
}

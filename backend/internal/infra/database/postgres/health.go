package postgres

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus represents database health status
type HealthStatus struct {
	Status        string    `json:"status"`         // "healthy", "unhealthy"
	ResponseTime  string    `json:"response_time"`  // e.g., "5ms"
	ActiveConns   int32     `json:"active_conns"`   // Current active connections
	IdleConns     int32     `json:"idle_conns"`     // Current idle connections
	TotalConns    int32     `json:"total_conns"`    // Total connections
	MaxConns      int32     `json:"max_conns"`      // Max connections allowed
	CheckedAt     time.Time `json:"checked_at"`     // When health check was performed
	Error         string    `json:"error,omitempty"` // Error message if unhealthy
}

// Health checks the health of the database connection
func (p *Pool) Health(ctx context.Context) *HealthStatus {
	start := time.Now()

	status := &HealthStatus{
		CheckedAt: start,
		Status:    "healthy",
	}

	// Ping database
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := p.Ping(pingCtx); err != nil {
		status.Status = "unhealthy"
		status.Error = fmt.Sprintf("ping failed: %v", err)
		status.ResponseTime = time.Since(start).String()
		return status
	}

	// Get pool stats
	stats := p.Stat()
	status.ActiveConns = stats.AcquiredConns()
	status.IdleConns = stats.IdleConns()
	status.TotalConns = stats.TotalConns()
	status.MaxConns = stats.MaxConns()
	status.ResponseTime = time.Since(start).String()

	// Check if connection pool is nearly exhausted
	if stats.AcquiredConns() >= stats.MaxConns()-2 {
		status.Status = "degraded"
		status.Error = "connection pool nearly exhausted"
	}

	return status
}

// IsHealthy returns true if the database is healthy
func (p *Pool) IsHealthy(ctx context.Context) bool {
	status := p.Health(ctx)
	return status.Status == "healthy"
}

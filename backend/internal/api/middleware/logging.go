package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggingConfig holds configuration for logging middleware
type LoggingConfig struct {
	AccessLogger *zerolog.Logger // Optional separate access logger
	SkipPaths    []string        // Paths to skip logging (e.g., /health)
}

// Logging middleware logs HTTP requests and responses
func Logging(cfg LoggingConfig) gin.HandlerFunc {
	// Use provided logger or default
	logger := log.Logger
	if cfg.AccessLogger != nil {
		logger = *cfg.AccessLogger
	}

	// Build skip map for faster lookup
	skipMap := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		// Skip if in skip list
		if skipMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		// Get request ID
		requestID := GetRequestID(c)

		// Log request start (DEBUG level)
		log.Debug().
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("→ Request started")

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Prepare log event
		event := logger.Info()

		// Use WARN for 4xx, ERROR for 5xx
		if statusCode >= 500 {
			event = logger.Error()
		} else if statusCode >= 400 {
			event = logger.Warn()
		}

		// Log response
		event.
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", statusCode).
			Int64("duration_ms", duration.Milliseconds()).
			Int("response_size", c.Writer.Size()).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent())

		// Add error if exists
		if len(c.Errors) > 0 {
			event.Str("error", c.Errors.String())
		}

		event.Msg("← Request completed")

		// Log slow requests (> 1s) with WARNING
		if duration > time.Second {
			log.Warn().
				Str("request_id", requestID).
				Str("method", c.Request.Method).
				Str("path", path).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("⚠️  Slow request detected")
		}
	}
}

// Recovery middleware with logging
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := GetRequestID(c)

				log.Error().
					Str("request_id", requestID).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Interface("panic", err).
					Msg("🚨 Panic recovered")

				// Return 500
				c.JSON(500, gin.H{
					"error": gin.H{
						"code":       "INTERNAL_SERVER_ERROR",
						"message":    "Internal server error",
						"request_id": requestID,
					},
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

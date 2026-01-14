package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader is the header name for request ID
const RequestIDHeader = "X-Request-ID"

// RequestIDKey is the context key for request ID
const RequestIDKey = "request_id"

// RequestID middleware adds a unique request ID to each request
// If X-Request-ID header exists, use it; otherwise generate a new one
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)

		if requestID == "" {
			// Generate new UUID
			requestID = uuid.New().String()
		}

		// Set in context
		c.Set(RequestIDKey, requestID)

		// Set in response header
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

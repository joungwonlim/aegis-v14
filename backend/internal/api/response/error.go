package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/wonny/aegis/v14/internal/api/middleware"
)

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Details   string          `json:"details,omitempty"`
	RequestID string          `json:"request_id"`
	Timestamp time.Time       `json:"timestamp"`
	Fields    []FieldError    `json:"fields,omitempty"`
}

// FieldError represents a field-level validation error
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error codes
const (
	// General errors
	ErrCodeInternalServer    = "INTERNAL_SERVER_ERROR"
	ErrCodeInvalidParameter  = "INVALID_PARAMETER"
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

	// Database errors
	ErrCodeDatabaseError       = "DATABASE_ERROR"
	ErrCodeDuplicateEntry      = "DUPLICATE_ENTRY"
	ErrCodeConstraintViolation = "CONSTRAINT_VIOLATION"

	// External API errors
	ErrCodeExternalAPIError   = "EXTERNAL_API_ERROR"
	ErrCodeExternalAPITimeout = "EXTERNAL_API_TIMEOUT"

	// Business logic errors
	ErrCodeBusinessRuleViolation = "BUSINESS_RULE_VIOLATION"
)

// Error sends an error response
func Error(c *gin.Context, statusCode int, code, message string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
		},
	}

	// Log error
	log.Error().
		Str("request_id", response.Error.RequestID).
		Str("error_code", code).
		Str("message", message).
		Int("status", statusCode).
		Msg("API error response")

	c.JSON(statusCode, response)
}

// ErrorWithDetails sends an error response with additional details
func ErrorWithDetails(c *gin.Context, statusCode int, code, message, details string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
		},
	}

	// Log error
	log.Error().
		Str("request_id", response.Error.RequestID).
		Str("error_code", code).
		Str("message", message).
		Str("details", details).
		Int("status", statusCode).
		Msg("API error response")

	c.JSON(statusCode, response)
}

// ValidationError sends a validation error response with field errors
func ValidationError(c *gin.Context, fields []FieldError) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:      ErrCodeValidation,
			Message:   "Request validation failed",
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
			Fields:    fields,
		},
	}

	// Log validation error
	log.Warn().
		Str("request_id", response.Error.RequestID).
		Str("error_code", ErrCodeValidation).
		Int("field_count", len(fields)).
		Msg("Validation error")

	c.JSON(http.StatusBadRequest, response)
}

// BadRequest sends a 400 Bad Request error
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, ErrCodeInvalidParameter, message)
}

// NotFound sends a 404 Not Found error
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, ErrCodeNotFound, message)
}

// Unauthorized sends a 401 Unauthorized error
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// Forbidden sends a 403 Forbidden error
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, ErrCodeForbidden, message)
}

// Conflict sends a 409 Conflict error
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, ErrCodeConflict, message)
}

// InternalError sends a 500 Internal Server Error
func InternalError(c *gin.Context, err error) {
	message := "An unexpected error occurred"
	details := ""

	if err != nil {
		details = err.Error()

		// Log the actual error with stack trace
		log.Error().
			Err(err).
			Str("request_id", middleware.GetRequestID(c)).
			Msg("Internal server error")
	}

	ErrorWithDetails(c, http.StatusInternalServerError, ErrCodeInternalServer, message, details)
}

// DatabaseError sends a database error response
func DatabaseError(c *gin.Context, err error) {
	message := "Database operation failed"
	details := ""

	if err != nil {
		details = err.Error()

		// Log the database error
		log.Error().
			Err(err).
			Str("request_id", middleware.GetRequestID(c)).
			Msg("Database error")
	}

	ErrorWithDetails(c, http.StatusInternalServerError, ErrCodeDatabaseError, message, details)
}

// ExternalAPIError sends an external API error response
func ExternalAPIError(c *gin.Context, serviceName string, err error) {
	message := "External service error"
	if serviceName != "" {
		message = serviceName + " service error"
	}

	details := ""
	if err != nil {
		details = err.Error()

		// Log the external API error
		log.Error().
			Err(err).
			Str("request_id", middleware.GetRequestID(c)).
			Str("service", serviceName).
			Msg("External API error")
	}

	ErrorWithDetails(c, http.StatusBadGateway, ErrCodeExternalAPIError, message, details)
}

// RateLimitExceeded sends a rate limit exceeded error
func RateLimitExceeded(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, ErrCodeRateLimitExceeded, "Rate limit exceeded")
}

// BusinessRuleViolation sends a business rule violation error
func BusinessRuleViolation(c *gin.Context, message string) {
	Error(c, http.StatusUnprocessableEntity, ErrCodeBusinessRuleViolation, message)
}

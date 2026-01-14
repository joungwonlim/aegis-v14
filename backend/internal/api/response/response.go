package response

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wonny/aegis/v14/internal/api/middleware"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Meta       Meta        `json:"meta"`
}

// Pagination represents pagination information
type Pagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	TotalPages int  `json:"total_pages"`
	TotalCount int  `json:"total_count"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// Meta represents metadata in response
type Meta struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
	Count     int       `json:"count,omitempty"`
}

// Success sends a successful response with data
func Success(c *gin.Context, data interface{}) {
	response := SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
		},
	}
	c.JSON(http.StatusOK, response)
}

// SuccessWithMessage sends a successful response with data and message
func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	response := SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
			Message:   message,
		},
	}
	c.JSON(http.StatusOK, response)
}

// SuccessList sends a successful response with list data and count
func SuccessList(c *gin.Context, data interface{}, count int) {
	response := SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
			Count:     count,
		},
	}
	c.JSON(http.StatusOK, response)
}

// SuccessWithPagination sends a successful response with pagination
func SuccessWithPagination(c *gin.Context, data interface{}, pagination *Pagination) {
	response := SuccessResponse{
		Data:       data,
		Pagination: pagination,
		Meta: Meta{
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
		},
	}
	c.JSON(http.StatusOK, response)
}

// Created sends a 201 Created response
func Created(c *gin.Context, data interface{}, message string) {
	response := SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: middleware.GetRequestID(c),
			Timestamp: time.Now(),
			Message:   message,
		},
	}
	c.JSON(http.StatusCreated, response)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// NewPagination creates a new Pagination object
func NewPagination(page, limit, totalCount int) *Pagination {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	totalPages := (totalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	return &Pagination{
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		TotalCount: totalCount,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// GetPaginationParams extracts pagination parameters from query string
func GetPaginationParams(c *gin.Context) (page int, limit int) {
	page = 1
	limit = 20

	if p := c.Query("page"); p != "" {
		if val, err := parseInt(p); err == nil && val > 0 {
			page = val
		}
	}

	if l := c.Query("limit"); l != "" {
		if val, err := parseInt(l); err == nil && val > 0 {
			limit = val
			if limit > 100 {
				limit = 100
			}
		}
	}

	return page, limit
}

// parseInt parses string to int
func parseInt(s string) (int, error) {
	var val int
	_, err := fmt.Sscanf(s, "%d", &val)
	return val, err
}

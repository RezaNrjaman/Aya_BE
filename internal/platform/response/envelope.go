package response

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PaginationMeta holds pagination metadata for list responses.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// SuccessResponse defines the standard success envelope.
type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// ListSuccessResponse defines the standard list envelope with pagination metadata.
type ListSuccessResponse struct {
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Meta    PaginationMeta `json:"meta"`
}

// ErrorDetail defines the error object within ErrorResponse.
type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// ErrorResponse defines the standard error envelope.
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// JSON sends a standard success response with HTTP 200.
func JSON(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// OK sends a standard success response with HTTP 200.
func OK(c *gin.Context, data any) {
	JSON(c, http.StatusOK, data)
}

// Created sends a standard success response with HTTP 201.
func Created(c *gin.Context, data any) {
	JSON(c, http.StatusCreated, data)
}

// List sends a standard list response with HTTP 200 and pagination meta.
// If data is nil, it is guaranteed to return an empty slice `[]`, never null.
func List(c *gin.Context, data any, page, perPage, total int) {
	if data == nil {
		data = []any{}
	}

	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	if page <= 0 {
		page = 1
	}

	totalPages := 0
	if total > 0 && perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	c.JSON(http.StatusOK, ListSuccessResponse{
		Success: true,
		Data:    data,
		Meta: PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Error sends a standard error response according to Panduan API Bersama.
func Error(c *gin.Context, statusCode int, code string, message string, fields map[string]string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	})
}

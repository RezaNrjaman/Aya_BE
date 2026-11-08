package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Standard error codes conforming to ARCHITECTURE.md and AGENT.md
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeValidationError     = "VALIDATION_ERROR"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
)

// AppError represents a structured domain or transport error.
type AppError struct {
	StatusCode int
	Code       string
	Message    string
	Fields     map[string]string
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Predefined error constructors
func BadRequest(message string) *AppError {
	if message == "" {
		message = "Permintaan tidak valid"
	}
	return &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       CodeBadRequest,
		Message:    message,
	}
}

func ValidationError(message string, fields map[string]string) *AppError {
	if message == "" {
		message = "Data tidak valid"
	}
	return &AppError{
		StatusCode: http.StatusUnprocessableEntity,
		Code:       CodeValidationError,
		Message:    message,
		Fields:     fields,
	}
}

func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Autentikasi gagal atau token tidak valid"
	}
	return &AppError{
		StatusCode: http.StatusUnauthorized,
		Code:       CodeUnauthorized,
		Message:    message,
	}
}

func Forbidden(message string) *AppError {
	if message == "" {
		message = "Anda tidak memiliki hak akses untuk tindakan ini"
	}
	return &AppError{
		StatusCode: http.StatusForbidden,
		Code:       CodeForbidden,
		Message:    message,
	}
}

func NotFound(message string) *AppError {
	if message == "" {
		message = "Data atau resource tidak ditemukan"
	}
	return &AppError{
		StatusCode: http.StatusNotFound,
		Code:       CodeNotFound,
		Message:    message,
	}
}

func Conflict(message string) *AppError {
	if message == "" {
		message = "Terjadi konflik data atau status tidak sesuai"
	}
	return &AppError{
		StatusCode: http.StatusConflict,
		Code:       CodeConflict,
		Message:    message,
	}
}

func InternalError(err error) *AppError {
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		Code:       CodeInternalServerError,
		Message:    "Terjadi kesalahan internal server",
		Err:        err,
	}
}

// HandleAppError writes an AppError into the gin response.
func HandleAppError(c *gin.Context, appErr *AppError) {
	Error(c, appErr.StatusCode, appErr.Code, appErr.Message, appErr.Fields)
}

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDContextKey = "request_id"

// RequestID middleware ensures every request has a unique correlation ID.
// If the incoming request has an X-Request-ID header, it is reused.
// Otherwise, a new UUID v4 is generated and set in response headers and context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Set(RequestIDContextKey, reqID)
		c.Header(RequestIDHeader, reqID)

		c.Next()
	}
}

// GetRequestID returns the request ID from gin context or an empty string.
func GetRequestID(c *gin.Context) string {
	if val, exists := c.Get(RequestIDContextKey); exists {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

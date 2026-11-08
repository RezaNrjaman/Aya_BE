package middleware

import (
	"time"

	"araya-business-hub/be/internal/platform/logging"
	"github.com/gin-gonic/gin"
)

// Logger returns a Gin middleware for structured request logging.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		reqID := GetRequestID(c)

		fullPath := path
		if rawQuery != "" {
			fullPath = path + "?" + rawQuery
		}

		errStr := ""
		if len(c.Errors) > 0 {
			errStr = c.Errors.String()
		}

		level := "INFO"
		if status >= 500 {
			level = "ERROR"
		} else if status >= 400 {
			level = "WARN"
		}

		logging.LogHTTP(logging.LogEntry{
			Level:     level,
			Message:   "HTTP Request",
			RequestID: reqID,
			Method:    c.Request.Method,
			Path:      fullPath,
			Status:    status,
			LatencyMs: latency,
			ClientIP:  c.ClientIP(),
			Error:     errStr,
		})
	}
}

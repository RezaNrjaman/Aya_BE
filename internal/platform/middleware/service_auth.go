package middleware

import (
	"crypto/subtle"

	"araya-business-hub/be/internal/platform/response"
	"github.com/gin-gonic/gin"
)

const (
	ServiceKeyHeader        = "X-Service-Key"
	ContextIsServiceCallKey = "is_service_call"
)

// ServiceAuthenticate checks the X-Service-Key header for inter-module calls without a user context.
func ServiceAuthenticate(configuredServiceKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if configuredServiceKey == "" {
			response.HandleAppError(c, response.Unauthorized("Kunci layanan antarmodul belum dikonfigurasi"))
			c.Abort()
			return
		}

		clientKey := c.GetHeader(ServiceKeyHeader)
		if clientKey == "" {
			response.HandleAppError(c, response.Unauthorized("Header X-Service-Key wajib disertakan"))
			c.Abort()
			return
		}

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(clientKey), []byte(configuredServiceKey)) != 1 {
			response.HandleAppError(c, response.Unauthorized("Kunci layanan antarmodul tidak valid"))
			c.Abort()
			return
		}

		c.Set(ContextIsServiceCallKey, true)
		c.Next()
	}
}

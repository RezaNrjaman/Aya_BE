package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS allows requests only from explicitly configured origins.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	origins := makeOriginSet(allowedOrigins)
	return func(c *gin.Context) { applyCORS(c, origins) }
}

func makeOriginSet(allowedOrigins []string) map[string]struct{} {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		value := strings.TrimSpace(origin)
		if value != "" && value != "*" {
			origins[value] = struct{}{}
		}
	}
	return origins
}

func applyCORS(c *gin.Context, origins map[string]struct{}) {
	origin := c.Request.Header.Get("Origin")
	if _, allowed := origins[origin]; allowed {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Vary", "Origin")
	}
	setCORSHeaders(c)
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}

func setCORSHeaders(c *gin.Context) {
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept, Authorization, X-Request-ID, X-Session-ID")
	c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID, X-Session-ID")
}

package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	tokens   float64
	lastSeen time.Time
	max      float64
	refill   float64 // tokens per second
}

// RateLimiter is a simple in-memory token bucket keyed by client IP and scope.
// It protects public endpoints (login, registration, status checks, activation)
// from brute force and enumeration (ARY-002). For multi-instance deployments
// replace with a shared store such as Redis.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*rateBucket)}
}

// Allow reports whether one request from the key may proceed.
func (rl *RateLimiter) Allow(key string, max int, window time.Duration) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok || now.Sub(b.lastSeen) > window {
		b = &rateBucket{tokens: float64(max), max: float64(max), refill: float64(max) / window.Seconds(), lastSeen: now}
		rl.buckets[key] = b
	}
	b.tokens += now.Sub(b.lastSeen).Seconds() * b.refill
	if b.tokens > b.max {
		b.tokens = b.max
	}
	b.lastSeen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimit returns Gin middleware that allows `max` requests per client IP
// within `window` for the wrapped route(s).
func RateLimit(rl *RateLimiter, max int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow("ip:"+c.ClientIP(), max, window) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMITED",
					"message": "Terlalu banyak percobaan. Silakan coba lagi nanti.",
				},
			})
			return
		}
		c.Next()
	}
}

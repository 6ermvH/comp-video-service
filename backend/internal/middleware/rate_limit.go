package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientRateState struct {
	windowStart time.Time
	count       int
}

// NewIPRateLimiter creates a fixed-window per-IP limiter.
func NewIPRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var (
		mu     sync.Mutex
		states = make(map[string]*clientRateState)
	)

	return func(c *gin.Context) {
		now := time.Now()
		ip := c.ClientIP()

		mu.Lock()
		state, ok := states[ip]
		if !ok || now.Sub(state.windowStart) >= window {
			state = &clientRateState{windowStart: now, count: 0}
			states[ip] = state
		}
		state.count++
		count := state.count
		windowStart := state.windowStart
		mu.Unlock()

		if count > limit {
			retryAfter := int(window.Seconds() - now.Sub(windowStart).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

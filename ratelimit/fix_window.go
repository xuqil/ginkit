package ratelimit

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"sync/atomic"
	"time"
)

// FixWindowLimiter Fix window limit middleware
type FixWindowLimiter struct {
	// timestamp is the starting timestamp of the window
	timestamp int64
	// interval specifies the size of the window
	interval int64
	// rate specifies the requests that can be released within a window
	rate int64
	// cnt is the number of requests for the window
	cnt int64
}

// NewFixWindowLimiter create *FixWindowLimiter. interval specifies the size of the window,
// rate specifies the requests that can be released within a window
func NewFixWindowLimiter(interval time.Duration, rate int64) *FixWindowLimiter {
	return &FixWindowLimiter{
		timestamp: time.Now().UnixNano(),
		interval:  interval.Nanoseconds(),
		rate:      rate,
	}
}

// BuildMiddleware is a function to build gin.HandlerFunc
func (f *FixWindowLimiter) BuildMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		current := time.Now().UnixNano()
		timestamp := atomic.LoadInt64(&f.timestamp)
		cnt := atomic.LoadInt64(&f.cnt)
		// New window
		if timestamp+f.interval < current {
			if atomic.CompareAndSwapInt64(&f.timestamp, timestamp, current) {
				atomic.CompareAndSwapInt64(&f.cnt, cnt, 0)
			}
		}

		cnt = atomic.AddInt64(&f.cnt, 1)
		// The number of requests has reached the threshold
		if cnt > f.rate {
			ctx.AbortWithStatus(http.StatusGatewayTimeout)
		}
		ctx.Next()
	}
}

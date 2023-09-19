package ratelimit

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// LeakyBucketLimiter Leaky bucket limit middleware
type LeakyBucketLimiter struct {
	producer *time.Ticker
	close    chan struct{}
}

// NewLeakyBucketLimiter create *LeakyBucketLimiter, Interval specifies the time interval for release requests.
func NewLeakyBucketLimiter(interval time.Duration) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		producer: time.NewTicker(interval),
		close:    make(chan struct{}),
	}
}

// BuildMiddleware is a function to build gin.HandlerFunc
func (l *LeakyBucketLimiter) BuildMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		select {
		case <-ctx.Request.Context().Done():
			ctx.AbortWithStatus(http.StatusGatewayTimeout)
		case <-l.close:
			ctx.Next()
		case <-l.producer.C:
			ctx.Next()
		}
	}
}

func (l *LeakyBucketLimiter) Close() error {
	l.producer.Stop()
	close(l.close)
	return nil
}

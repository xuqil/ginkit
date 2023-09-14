package ratelimit

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// TokenBucketLimiter Token bucket limit middleware
type TokenBucketLimiter struct {
	tokens chan struct{}
	close  chan struct{}
}

// NewTokenBucketLimiter create *TokenBucketLimiter. capacity specifies the capacity of the bucket,
// interval specifies the time to generate the token.
func NewTokenBucketLimiter(capacity uint64, interval time.Duration) *TokenBucketLimiter {
	tokenCh := make(chan struct{}, capacity)
	closeCh := make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case tokenCh <- struct{}{}:
				default:
				}
			case <-closeCh:
				return
			}
		}
	}()

	return &TokenBucketLimiter{
		tokens: tokenCh,
		close:  closeCh,
	}
}

// BuildMiddleware is a function to build gin.HandlerFunc
func (t *TokenBucketLimiter) BuildMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		select {
		case <-t.close:
			ctx.AbortWithStatus(http.StatusGatewayTimeout)
		case <-ctx.Request.Context().Done():
			ctx.AbortWithStatus(http.StatusGatewayTimeout)
		case <-t.tokens:
			ctx.Next()
		}
	}
}

func (t *TokenBucketLimiter) Close() error {
	close(t.close)
	return nil
}

package ratelimit

import (
	"container/list"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
	"time"
)

type SlideWindowLimiter struct {
	queue    *list.List
	interval int64
	rate     int
	mutex    sync.Mutex
}

func NewSlideWindowLimiter(interval time.Duration, rate int) *SlideWindowLimiter {
	return &SlideWindowLimiter{
		queue:    list.New(),
		interval: interval.Nanoseconds(),
		rate:     rate,
	}
}

// BuildMiddleware is a function to build gin.HandlerFunc
func (s *SlideWindowLimiter) BuildMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		now := time.Now().UnixNano()
		boundary := now - s.interval

		s.mutex.Lock()
		length := s.queue.Len()
		// Fast path
		if length < s.rate {
			s.queue.PushBack(now)
			s.mutex.Unlock()
			ctx.Next()
			return
		}

		// Slow path
		timestamp := s.queue.Front()
		for timestamp != nil && timestamp.Value.(int64) < boundary {
			s.queue.Remove(timestamp)
			timestamp = s.queue.Front()
		}
		length = s.queue.Len()
		s.mutex.Unlock()
		// The number of requests has reached the threshold
		if length >= s.rate {
			ctx.AbortWithStatus(http.StatusGatewayTimeout)
		}
		s.queue.PushBack(now)
		ctx.Next()
	}
}

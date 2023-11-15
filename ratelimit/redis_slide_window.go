package ratelimit

import (
	"context"
	_ "embed"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"net/http"
	"time"
)

//go:embed lua/slide_window.lua
var luaSlideWindow string

type RedisSlideWindowLimiter struct {
	client   redis.Cmdable
	key      string
	interval time.Duration
	rate     int64
}

func NewRedisSlideWindowLimiter(client redis.Cmdable, key string,
	interval time.Duration, rate int64) *RedisSlideWindowLimiter {
	return &RedisSlideWindowLimiter{
		client:   client,
		key:      key,
		interval: interval,
		rate:     rate,
	}
}

// BuildMiddleware is a function to build gin.HandlerFunc
func (f *RedisSlideWindowLimiter) BuildMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		limit, err := f.limit(ctx.Request.Context())
		if err != nil {
			return
		}
		if limit {
			ctx.AbortWithStatus(http.StatusTooManyRequests)
		}
		ctx.Next()
	}
}

func (f *RedisSlideWindowLimiter) limit(ctx context.Context) (bool, error) {
	return f.client.Eval(ctx, luaSlideWindow, []string{f.key}, f.interval.Milliseconds(),
		f.rate, time.Now().UnixMilli()).Bool()
}

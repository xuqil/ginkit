package ratelimit

import (
	"context"
	_ "embed"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"net/http"
	"time"
)

//go:embed lua/fix_window.lua
var luaFixWindow string

type RedisFixWindowLimiter struct {
	client   redis.Cmdable
	key      string
	interval time.Duration
	rate     int64
}

func NewRedisFixWindowLimiter(client redis.Cmdable, key string,
	interval time.Duration, rate int64) *RedisFixWindowLimiter {
	return &RedisFixWindowLimiter{
		client:   client,
		key:      key,
		interval: interval,
		rate:     rate,
	}
}

// BuildMiddleware is a function to build gin.HandlerFunc
func (f *RedisFixWindowLimiter) BuildMiddleware() gin.HandlerFunc {
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

func (f *RedisFixWindowLimiter) limit(ctx context.Context) (bool, error) {
	return f.client.Eval(ctx, luaFixWindow, []string{f.key}, f.interval.Milliseconds(), f.rate).Bool()
}

//go:build e2e

package ratelimit

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRedisSlideWindowLimiter_e2e_BuildServerInterceptor(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	mdl := NewRedisSlideWindowLimiter(rdb, "slide-limit", time.Second*3, 1).BuildMiddleware()
	cnt := 0
	handler := func(ctx *gin.Context) {
		cnt++
		ctx.Status(http.StatusOK)
	}

	server := gin.Default()
	server.Use(mdl)
	server.GET("/", handler)

	req, _ := http.NewRequest("GET", "/", nil)
	req = req.WithContext(context.Background())
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Reaching threshold
	req = req.WithContext(context.Background())
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Reset
	time.Sleep(time.Second * 3)
	req = req.WithContext(context.Background())
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

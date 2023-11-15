package ratelimit

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLeakyLeakyBucketLimiter_BuildMiddleware(t *testing.T) {
	testCases := []struct {
		name    string
		b       func() *LeakyBucketLimiter
		handler func(ctx *gin.Context)
		ctx     func() context.Context

		wantStatus int
	}{
		{
			name: "stopped",
			b: func() *LeakyBucketLimiter {
				producer := time.NewTicker(time.Second)
				res := &LeakyBucketLimiter{
					producer: producer,
					close:    make(chan struct{}),
				}
				_ = res.Close()
				return res
			},
			ctx: func() context.Context {
				return context.Background()
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "context canceled",
			b: func() *LeakyBucketLimiter {
				return &LeakyBucketLimiter{
					producer: time.NewTicker(time.Second * 3),
					close:    make(chan struct{}),
				}
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "get tokens",
			b: func() *LeakyBucketLimiter {
				ch := make(chan struct{}, 1)
				ch <- struct{}{}
				return &LeakyBucketLimiter{
					producer: time.NewTicker(time.Second * 1),
					close:    make(chan struct{}),
				}
			},
			ctx: func() context.Context {
				return context.Background()
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mdl := tc.b().BuildMiddleware()
			server := gin.Default()
			server.Use(mdl)
			server.GET("/", func(ctx *gin.Context) {
				ctx.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/", nil)
			req = req.WithContext(tc.ctx())
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestLeakyBucketLimiter_Leaky(t *testing.T) {
	limiter := NewLeakyBucketLimiter(time.Second * 2)
	defer func() {
		_ = limiter.Close()
	}()
	mdl := limiter.BuildMiddleware()
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

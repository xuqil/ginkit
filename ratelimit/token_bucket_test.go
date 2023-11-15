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

func TestTokenBucketLimiter_BuildMiddleware(t *testing.T) {
	testCases := []struct {
		name    string
		b       func() *TokenBucketLimiter
		handler func(ctx *gin.Context)
		ctx     func() context.Context

		wantStatus int
	}{
		{
			name: "closed",
			b: func() *TokenBucketLimiter {
				closeChan := make(chan struct{})
				close(closeChan)
				return &TokenBucketLimiter{
					tokens: make(chan struct{}),
					close:  closeChan,
				}
			},
			ctx: func() context.Context {
				return context.Background()
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "context canceled",
			b: func() *TokenBucketLimiter {
				return &TokenBucketLimiter{
					tokens: make(chan struct{}),
					close:  make(chan struct{}),
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
			b: func() *TokenBucketLimiter {
				ch := make(chan struct{}, 1)
				ch <- struct{}{}
				return &TokenBucketLimiter{
					tokens: ch,
					close:  make(chan struct{}),
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

func TestTokenBucketLimiter_Tokens(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, time.Second*2)
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

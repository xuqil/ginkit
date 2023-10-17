package ratelimit

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRedisFixWindowLimiter_e2e_BuildMiddleware(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	mdl := NewRedisFixWindowLimiter(rdb, "fix-limit", time.Second*3, 1).BuildMiddleware()
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
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)

	// Reset
	time.Sleep(time.Second * 3)
	req = req.WithContext(context.Background())
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFixWindowLimiter_LimitUnary(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	testCases := []struct {
		name     string
		key      string
		rate     int64
		interval time.Duration

		before func(t *testing.T)
		after  func(t *testing.T)

		wantLimit bool
		wantErr   error
	}{
		{
			name:     "init",
			key:      "my-limit",
			rate:     1,
			interval: time.Minute,
			before:   func(t *testing.T) {},
			after: func(t *testing.T) {
				val, err := rdb.Get(context.Background(), "my-limit").Result()
				require.NoError(t, err)
				assert.Equal(t, "1", val)
				_, err = rdb.Del(context.Background(), "my-limit").Result()
				require.NoError(t, err)
			},
		},
		{
			name:      "init but limit",
			key:       "my-limit",
			rate:      0,
			wantLimit: true,
			interval:  time.Minute,
			before:    func(t *testing.T) {},
			after: func(t *testing.T) {
				_, err := rdb.Get(context.Background(), "my-limit").Result()
				require.Equal(t, redis.Nil, err)
			},
		},
		{
			name:      "limit",
			key:       "my-limit",
			rate:      5,
			wantLimit: true,
			interval:  time.Minute,
			before: func(t *testing.T) {
				val, err := rdb.Set(context.Background(), "my-limit", 5, time.Minute).Result()
				require.NoError(t, err)
				assert.Equal(t, "OK", val)
			},
			after: func(t *testing.T) {
				val, err := rdb.Get(context.Background(), "my-limit").Result()
				require.NoError(t, err)
				assert.Equal(t, "5", val)
				_, _ = rdb.Del(context.Background(), "my-limit").Result()
			},
		},
		{
			name:     "window shift",
			key:      "my-limit",
			rate:     5,
			interval: time.Minute,
			before: func(t *testing.T) {
				val, err := rdb.Set(context.Background(), "my-limit", 5, time.Second).Result()
				require.NoError(t, err)
				assert.Equal(t, "OK", val)
				time.Sleep(time.Second * 2)
			},
			after: func(t *testing.T) {
				val, err := rdb.Get(context.Background(), "my-limit").Result()
				require.NoError(t, err)
				assert.Equal(t, "1", val)
				_, _ = rdb.Del(context.Background(), "my-limit").Result()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			defer tc.after(t)
			l := NewRedisFixWindowLimiter(rdb, tc.key, tc.interval, tc.rate)
			limit, err := l.limit(context.Background())
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantLimit, limit)
		})
	}
}

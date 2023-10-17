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

func TestFixWindowLimiter_BuildMiddleware(t *testing.T) {
	limiter := NewFixWindowLimiter(time.Second*3, 1)
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

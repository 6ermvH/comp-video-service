package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAuthMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth("secret"))
	r.GET("/p", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "admin-1", "csrf": "t"})
	signed, _ := token.SignedString([]byte("secret"))

	r := gin.New()
	r.Use(RequireAuth("secret"))
	r.GET("/p", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestRateLimiterBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewIPRateLimiter(1, time.Minute))
	r.GET("/p", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/p", nil))
	if w1.Code != http.StatusNoContent {
		t.Fatalf("expected first 204, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/p", nil))
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second 429, got %d", w2.Code)
	}
}

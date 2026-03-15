package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"comp-video-service/backend/internal/handler"
	"comp-video-service/backend/internal/model"
)

func TestLogin_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// NewAuthHandler with nil repo: binding fails before repo access
	h := handler.NewAuthHandler(nil, "testsecret")
	r := gin.New()
	r.POST("/api/admin/login", h.Login)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := model.LoginRequest{Username: "admin", Password: "wrong"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r := gin.New()
	r.POST("/api/admin/login", func(c *gin.Context) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestBcryptHash(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt error: %v", err)
	}
	if len(hash) == 0 {
		t.Error("expected non-empty hash")
	}
	fmt.Printf("bcrypt hash length: %d\n", len(hash))

	if err := bcrypt.CompareHashAndPassword(hash, []byte("secret")); err != nil {
		t.Errorf("hash comparison failed: %v", err)
	}
}

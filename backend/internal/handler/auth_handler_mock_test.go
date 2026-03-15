package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"comp-video-service/backend/internal/model"
)

type mockAdminAuthRepo struct {
	getFn func(context.Context, string) (*model.Admin, error)
}

func (m *mockAdminAuthRepo) GetByUsername(ctx context.Context, username string) (*model.Admin, error) {
	return m.getFn(ctx, username)
}

func TestAuthHandlerLoginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	h := NewAuthHandler(&mockAdminAuthRepo{getFn: func(context.Context, string) (*model.Admin, error) {
		return &model.Admin{ID: uuid.New(), Username: "admin", PasswordHash: string(hash)}, nil
	}}, "jwt-secret")

	r := gin.New()
	r.POST("/login", h.Login)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", mustJSON(t, map[string]string{"username": "admin", "password": "secret"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandlerLoginInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(&mockAdminAuthRepo{getFn: func(context.Context, string) (*model.Admin, error) {
		return nil, errors.New("not found")
	}}, "jwt-secret")

	r := gin.New()
	r.POST("/login", h.Login)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", mustJSON(t, map[string]string{"username": "admin", "password": "secret"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

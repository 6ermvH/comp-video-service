package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/service"
)

type mockSessionService struct {
	startFn    func(context.Context, *model.StartSessionRequest) (*service.SessionStartResult, error)
	nextTaskFn func(context.Context, string) (*model.TaskPayload, error)
	completeFn func(context.Context, string) (*service.SessionCompleteResult, error)
}

func (m *mockSessionService) Start(ctx context.Context, req *model.StartSessionRequest) (*service.SessionStartResult, error) {
	return m.startFn(ctx, req)
}
func (m *mockSessionService) NextTask(ctx context.Context, token string) (*model.TaskPayload, error) {
	return m.nextTaskFn(ctx, token)
}
func (m *mockSessionService) Complete(ctx context.Context, token string) (*service.SessionCompleteResult, error) {
	return m.completeFn(ctx, token)
}

func TestSessionHandlerStartSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSessionHandler(&mockSessionService{
		startFn: func(context.Context, *model.StartSessionRequest) (*service.SessionStartResult, error) {
			return &service.SessionStartResult{SessionToken: "abc", Assigned: 1}, nil
		},
	})
	r := gin.New()
	r.POST("/api/session/start", h.Start)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/session/start", mustJSON(t, map[string]any{"study_id": uuid.New()}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSessionHandlerNextTaskNoRows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSessionHandler(&mockSessionService{
		nextTaskFn: func(context.Context, string) (*model.TaskPayload, error) {
			return nil, pgx.ErrNoRows
		},
	})
	r := gin.New()
	r.GET("/api/session/:token/next-task", h.NextTask)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/session/token123/next-task", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestSessionHandlerCompleteError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSessionHandler(&mockSessionService{
		completeFn: func(context.Context, string) (*service.SessionCompleteResult, error) {
			return nil, errors.New("boom")
		},
	})
	r := gin.New()
	r.POST("/api/session/:token/complete", h.Complete)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/session/token123/complete", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSessionHandlerStartBadRequestAndServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSessionHandler(&mockSessionService{
		startFn: func(context.Context, *model.StartSessionRequest) (*service.SessionStartResult, error) {
			return nil, errors.New("boom")
		},
	})
	r := gin.New()
	r.POST("/api/session/start", h.Start)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/session/start", mustJSON(t, map[string]any{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bind, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/session/start", mustJSON(t, map[string]any{"study_id": uuid.New()}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for service error, got %d", w.Code)
	}
}

func TestSessionHandlerNextTaskBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSessionHandler(&mockSessionService{
		nextTaskFn: func(context.Context, string) (*model.TaskPayload, error) {
			return nil, errors.New("boom")
		},
	})
	r := gin.New()
	r.GET("/api/session/:token/next-task", h.NextTask)
	r.GET("/api/session/next-task", h.NextTask)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/session/next-task", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty token, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/session/abc/next-task", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for service error, got %d", w.Code)
	}
}

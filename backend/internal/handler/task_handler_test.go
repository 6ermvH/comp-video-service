package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"comp-video-service/backend/internal/model"
)

type mockTaskService struct {
	saveFn func(context.Context, uuid.UUID, *model.TaskResponseRequest) (*model.Response, error)
}

func (m *mockTaskService) SaveResponse(ctx context.Context, id uuid.UUID, req *model.TaskResponseRequest) (*model.Response, error) {
	return m.saveFn(ctx, id, req)
}

type mockPairRepo struct {
	getFn func(context.Context, uuid.UUID) (*model.PairPresentation, error)
}

func (m *mockPairRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.PairPresentation, error) {
	return m.getFn(ctx, id)
}

type mockInteractionRepo struct {
	createFn func(context.Context, *model.InteractionLog) (*model.InteractionLog, error)
}

func (m *mockInteractionRepo) Create(ctx context.Context, e *model.InteractionLog) (*model.InteractionLog, error) {
	return m.createFn(ctx, e)
}

func TestTaskHandlerSubmitResponseConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(
		&mockTaskService{saveFn: func(context.Context, uuid.UUID, *model.TaskResponseRequest) (*model.Response, error) {
			return nil, &pgconn.PgError{Code: "23505"}
		}},
		&mockPairRepo{},
		&mockInteractionRepo{},
	)
	r := gin.New()
	r.POST("/api/task/:id/response", h.SubmitResponse)

	id := uuid.New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/response", mustJSON(t, map[string]any{"choice": "left"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestTaskHandlerLogEventSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pid := uuid.New()
	h := NewTaskHandler(
		&mockTaskService{},
		&mockPairRepo{getFn: func(context.Context, uuid.UUID) (*model.PairPresentation, error) {
			return &model.PairPresentation{ID: pid, ParticipantID: uuid.New()}, nil
		}},
		&mockInteractionRepo{createFn: func(context.Context, *model.InteractionLog) (*model.InteractionLog, error) {
			return &model.InteractionLog{ID: uuid.New(), EventTS: time.Now()}, nil
		}},
	)

	r := gin.New()
	r.POST("/api/task/:id/event", h.LogEvent)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/task/"+pid.String()+"/event", mustJSON(t, map[string]any{"event_type": "replay"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestTaskHandlerLogEventTaskNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(&mockTaskService{}, &mockPairRepo{getFn: func(context.Context, uuid.UUID) (*model.PairPresentation, error) {
		return nil, errors.New("not found")
	}}, &mockInteractionRepo{})
	r := gin.New()
	r.POST("/api/task/:id/event", h.LogEvent)

	id := uuid.New()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/event", mustJSON(t, map[string]any{"event_type": "x"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTaskHandlerSubmitResponseBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(
		&mockTaskService{saveFn: func(context.Context, uuid.UUID, *model.TaskResponseRequest) (*model.Response, error) {
			return nil, errors.New("boom")
		}},
		&mockPairRepo{},
		&mockInteractionRepo{},
	)
	r := gin.New()
	r.POST("/api/task/:id/response", h.SubmitResponse)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/task/invalid/response", mustJSON(t, map[string]any{"choice": "left"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid id, got %d", w.Code)
	}

	id := uuid.New()
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/response", mustJSON(t, map[string]any{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 bind, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/response", mustJSON(t, map[string]any{"choice": "left"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 service error, got %d", w.Code)
	}
}

func TestTaskHandlerLogEventBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTaskHandler(
		&mockTaskService{},
		&mockPairRepo{getFn: func(context.Context, uuid.UUID) (*model.PairPresentation, error) {
			return &model.PairPresentation{ID: uuid.New(), ParticipantID: uuid.New()}, nil
		}},
		&mockInteractionRepo{createFn: func(context.Context, *model.InteractionLog) (*model.InteractionLog, error) {
			return nil, errors.New("boom")
		}},
	)
	r := gin.New()
	r.POST("/api/task/:id/event", h.LogEvent)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/task/invalid/event", mustJSON(t, map[string]any{"event_type": "x"})))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid id, got %d", w.Code)
	}

	id := uuid.New()
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/event", mustJSON(t, map[string]any{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 bind, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/event", mustJSON(t, map[string]any{"event_type": ""}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 empty event_type, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/task/"+id.String()+"/event", mustJSON(t, map[string]any{"event_type": "ok"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 create error, got %d", w.Code)
	}
}

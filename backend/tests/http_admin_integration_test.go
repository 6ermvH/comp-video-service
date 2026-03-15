//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"comp-video-service/backend/internal/handler"
	"comp-video-service/backend/internal/middleware"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/service"
)

func TestAdminHTTPFlow_LoginCSRFAndCRUD(t *testing.T) {
	db := mustOpenDB(t)
	ctx := context.Background()
	seedAdmin(t, ctx, db, "admin_http_flow", "secret123")

	router := buildAdminRouter(db, "test-secret")

	token, csrf := loginAdmin(t, router, "admin_http_flow", "secret123")
	if token == "" || csrf == "" {
		t.Fatal("expected non-empty token and csrf")
	}

	// GET protected endpoint without csrf (allowed for GET)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/studies", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET studies expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// POST without csrf should fail
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studies", strings.NewReader(`{"name":"Smoke Study","effect_type":"flooding"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST studies without csrf expected 403, got %d", w.Code)
	}

	// POST with csrf should succeed
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studies", strings.NewReader(`{"name":"Smoke Study","effect_type":"flooding"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CSRF-Token", csrf)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST studies expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created study: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected created study id")
	}

	// Create group for study
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/admin/studies/"+created.ID.String()+"/groups", strings.NewReader(`{"name":"Group Smoke","priority":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CSRF-Token", csrf)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST study groups expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List groups by study
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/studies/"+created.ID.String()+"/groups", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET study groups expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Group Smoke") {
		t.Fatalf("expected created group in groups list, got %s", w.Body.String())
	}

	// PATCH study status with csrf
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/admin/studies/"+created.ID.String(), strings.NewReader(`{"status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CSRF-Token", csrf)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH study status expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Analytics overview
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/analytics/overview", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("analytics overview expected 200, got %d", w.Code)
	}

	// Export CSV
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/admin/export/csv", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("export csv expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "response_id") {
		t.Fatalf("export csv does not contain expected header, got: %s", w.Body.String())
	}
}

func buildAdminRouter(db *pgxpool.Pool, jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	adminRepo := repository.NewAdminRepository(db)
	studyRepo := repository.NewStudyRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	sourceItemRepo := repository.NewSourceItemRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	responseRepo := repository.NewResponseRepository(db)

	authH := handler.NewAuthHandler(adminRepo, jwtSecret)
	studySvc := service.NewStudyService(studyRepo, groupRepo, sourceItemRepo, videoRepo)
	assetSvc := service.NewAssetService(videoRepo, nil)
	analyticsSvc := service.NewAnalyticsService(db, responseRepo)
	qcSvc := service.NewQCService(responseRepo, participantRepo)
	exportSvc := service.NewExportService(db)
	adminH := handler.NewAdminHandler(studySvc, assetSvc, analyticsSvc, qcSvc, exportSvc)

	api := r.Group("/api")
	api.POST("/admin/login", authH.Login)

	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.RequireAuth(jwtSecret))
	adminGroup.Use(middleware.RequireCSRF())
	{
		adminGroup.GET("/studies", adminH.ListStudies)
		adminGroup.POST("/studies", adminH.CreateStudy)
		adminGroup.PATCH("/studies/:id", adminH.PatchStudyStatus)
		adminGroup.GET("/studies/:id/groups", adminH.ListGroups)
		adminGroup.POST("/studies/:id/groups", adminH.CreateGroup)
		adminGroup.GET("/analytics/overview", adminH.AnalyticsOverview)
		adminGroup.GET("/export/csv", adminH.ExportCSV)
	}

	return r
}

func seedAdmin(t *testing.T, ctx context.Context, db *pgxpool.Pool, username, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	_, err = db.Exec(ctx, `
		INSERT INTO admins (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username)
		DO UPDATE SET password_hash = EXCLUDED.password_hash`, username, string(hash))
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
}

func loginAdmin(t *testing.T, router *gin.Engine, username, password string) (string, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var out struct {
		Token     string `json:"token"`
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return out.Token, out.CSRFToken
}

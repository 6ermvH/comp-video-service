//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/handler"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/service"
)

func TestRespondentHTTPFlow_StartNextTaskComplete_NoTasks(t *testing.T) {
	db := mustOpenDB(t)
	ctx := context.Background()
	studyID := mustCreateStudy(t, ctx, db, "http-respondent-start", "active")

	router := buildRespondentRouter(db)

	startPayload, _ := json.Marshal(map[string]any{
		"study_id":    studyID,
		"device_type": "desktop",
		"browser":     "chrome",
		"role":        "general_viewer",
		"experience":  "limited",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/session/start", bytes.NewReader(startPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session start expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var startResp struct {
		SessionToken string `json:"session_token"`
		Assigned     int    `json:"assigned"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if startResp.SessionToken == "" {
		t.Fatal("expected non-empty session_token")
	}
	if startResp.Assigned != 0 {
		t.Fatalf("expected assigned=0, got %d", startResp.Assigned)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/session/"+startResp.SessionToken+"/next-task", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("next-task expected 204, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/session/"+startResp.SessionToken+"/complete", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("session complete expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var completeResp struct {
		CompletionCode string `json:"completion_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &completeResp); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	if completeResp.CompletionCode == "" {
		t.Fatal("expected completion code")
	}
}

func TestRespondentHTTPFlow_TaskResponseAndEvent(t *testing.T) {
	db := mustOpenDB(t)
	ctx := context.Background()
	studyID := mustCreateStudy(t, ctx, db, "http-respondent-task", "active")
	groupID := mustCreateGroup(t, ctx, db, studyID, "http-group")
	sourceItemID := mustCreateSourceItem(t, ctx, db, studyID, groupID)
	leftID := mustCreateVideoAsset(t, ctx, db, sourceItemID, "baseline", "videos/http-left.mp4")
	rightID := mustCreateVideoAsset(t, ctx, db, sourceItemID, "candidate", "videos/http-right.mp4")
	participantID, _ := mustCreateParticipant(t, ctx, db, studyID)
	pairID := mustCreatePairPresentation(t, ctx, db, participantID, sourceItemID, leftID, rightID)

	router := buildRespondentRouter(db)

	responsePayload, _ := json.Marshal(map[string]any{
		"choice":           "left",
		"reason_codes":     []string{"realism"},
		"confidence":       5,
		"response_time_ms": 3100,
		"replay_count":     1,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/task/"+pairID.String()+"/response", bytes.NewReader(responsePayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("task response expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var respBody struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	if respBody.ID == uuid.Nil {
		t.Fatal("expected created response id")
	}

	eventPayload, _ := json.Marshal(map[string]any{
		"event_type":   "replay_clicked",
		"payload_json": map[string]any{"count": 1},
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/task/"+pairID.String()+"/event", bytes.NewReader(eventPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("task event expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func buildRespondentRouter(db *pgxpool.Pool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	studyRepo := repository.NewStudyRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	sourceItemRepo := repository.NewSourceItemRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	pairRepo := repository.NewPairPresentationRepository(db)
	responseRepo := repository.NewResponseRepository(db)
	interactionRepo := repository.NewInteractionLogRepository(db)

	assignmentSvc := service.NewAssignmentService(sourceItemRepo, groupRepo, videoRepo, pairRepo)
	qcSvc := service.NewQCService(responseRepo, participantRepo)
	sessionSvc := service.NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, nil)

	sessionH := handler.NewSessionHandler(sessionSvc)
	taskH := handler.NewTaskHandler(sessionSvc, pairRepo, interactionRepo)

	api := r.Group("/api")
	api.POST("/session/start", sessionH.Start)
	api.GET("/session/:token/next-task", sessionH.NextTask)
	api.POST("/session/:token/complete", sessionH.Complete)
	api.POST("/task/:id/response", taskH.SubmitResponse)
	api.POST("/task/:id/event", taskH.LogEvent)

	return r
}

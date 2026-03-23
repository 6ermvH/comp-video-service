package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"comp-video-service/backend/internal/service"
)


func setupChunkedRouter(h *ChunkedUploadHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/uploads/init", h.InitUpload)
	r.POST("/uploads/:upload_id/chunks/:index", h.UploadChunk)
	r.POST("/uploads/:upload_id/complete", h.CompleteUpload)
	r.DELETE("/uploads/:upload_id", h.AbortUpload)
	return r
}

func initUpload(t *testing.T, r *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/uploads/init", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("InitUpload: expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("InitUpload: failed to parse response: %v", err)
	}
	id, ok := body["upload_id"]
	if !ok || id == "" {
		t.Fatal("InitUpload: missing upload_id in response")
	}
	return id
}

func uploadChunk(r *gin.Engine, uploadID string, index int, data []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/uploads/%s/chunks/%d", uploadID, index), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/octet-stream")
	r.ServeHTTP(w, req)
	return w
}

func completeUpload(r *gin.Engine, uploadID, name, effectType string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"name": name, "effect_type": effectType})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/uploads/%s/complete", uploadID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// TestChunkedUpload_FullFlow tests the happy path: init -> 2 chunks -> complete.
func TestChunkedUpload_FullFlow(t *testing.T) {
	svc := &mockImportService{
		importFn: func(_ context.Context, req service.ImportArchiveRequest) (*service.ImportArchiveResult, error) {
			return &service.ImportArchiveResult{
				GroupsCreated:  1,
				PairsCreated:   2,
				VideosUploaded: 4,
			}, nil
		},
	}
	h := NewChunkedUploadHandler(svc)
	r := setupChunkedRouter(h)

	uploadID := initUpload(t, r)

	chunk0 := []byte("chunk-data-0")
	w := uploadChunk(r, uploadID, 0, chunk0)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadChunk 0: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp0 map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp0)
	if int(resp0["index"].(float64)) != 0 {
		t.Errorf("UploadChunk 0: expected index 0, got %v", resp0["index"])
	}
	if int(resp0["received_bytes"].(float64)) != len(chunk0) {
		t.Errorf("UploadChunk 0: expected received_bytes %d, got %v", len(chunk0), resp0["received_bytes"])
	}

	chunk1 := []byte("chunk-data-1")
	w = uploadChunk(r, uploadID, 1, chunk1)
	if w.Code != http.StatusOK {
		t.Fatalf("UploadChunk 1: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp1 map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp1)
	if int(resp1["index"].(float64)) != 1 {
		t.Errorf("UploadChunk 1: expected index 1, got %v", resp1["index"])
	}

	w = completeUpload(r, uploadID, "My Study", "blur")
	if w.Code != http.StatusCreated {
		t.Fatalf("CompleteUpload: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var result service.ImportArchiveResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("CompleteUpload: failed to parse response: %v", err)
	}
	if result.GroupsCreated != 1 || result.PairsCreated != 2 || result.VideosUploaded != 4 {
		t.Errorf("CompleteUpload: unexpected result %+v", result)
	}
}

// TestChunkedUpload_NotFound tests uploading a chunk to a nonexistent upload_id.
func TestChunkedUpload_NotFound(t *testing.T) {
	h := NewChunkedUploadHandler(&mockImportService{})
	r := setupChunkedRouter(h)

	w := uploadChunk(r, "nonexistent-id", 0, []byte("data"))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestChunkedUpload_DuplicateChunk tests uploading the same chunk index twice.
func TestChunkedUpload_DuplicateChunk(t *testing.T) {
	h := NewChunkedUploadHandler(&mockImportService{})
	r := setupChunkedRouter(h)

	uploadID := initUpload(t, r)

	w := uploadChunk(r, uploadID, 0, []byte("first"))
	if w.Code != http.StatusOK {
		t.Fatalf("first upload: expected 200, got %d", w.Code)
	}

	w = uploadChunk(r, uploadID, 0, []byte("duplicate"))
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate chunk: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// TestChunkedUpload_NoChunks tests completing an upload with no chunks.
func TestChunkedUpload_NoChunks(t *testing.T) {
	h := NewChunkedUploadHandler(&mockImportService{})
	r := setupChunkedRouter(h)

	uploadID := initUpload(t, r)

	w := completeUpload(r, uploadID, "My Study", "blur")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestChunkedUpload_MissingMetadata tests completing an upload without required name field.
func TestChunkedUpload_MissingMetadata(t *testing.T) {
	h := NewChunkedUploadHandler(&mockImportService{})
	r := setupChunkedRouter(h)

	uploadID := initUpload(t, r)
	_ = uploadChunk(r, uploadID, 0, []byte("data"))

	// Missing name
	w := completeUpload(r, uploadID, "", "blur")
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing name: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestChunkedUpload_AbortUpload tests aborting an upload and verifying it's gone.
func TestChunkedUpload_AbortUpload(t *testing.T) {
	h := NewChunkedUploadHandler(&mockImportService{})
	r := setupChunkedRouter(h)

	uploadID := initUpload(t, r)
	_ = uploadChunk(r, uploadID, 0, []byte("data"))

	// Abort
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/uploads/%s", uploadID), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("AbortUpload: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Uploading another chunk should now return 404
	w = uploadChunk(r, uploadID, 1, []byte("more-data"))
	if w.Code != http.StatusNotFound {
		t.Errorf("after abort: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// TestChunkedUpload_ImportError tests that an import service error returns 500.
func TestChunkedUpload_ImportError(t *testing.T) {
	svc := &mockImportService{
		importFn: func(_ context.Context, _ service.ImportArchiveRequest) (*service.ImportArchiveResult, error) {
			return nil, errors.New("import failed: storage unavailable")
		},
	}
	h := NewChunkedUploadHandler(svc)
	r := setupChunkedRouter(h)

	uploadID := initUpload(t, r)
	_ = uploadChunk(r, uploadID, 0, []byte("data"))

	w := completeUpload(r, uploadID, "My Study", "blur")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

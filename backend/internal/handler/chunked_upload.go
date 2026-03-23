package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"comp-video-service/backend/internal/service"
)

const (
	maxUploadSize  = int64(1 << 30) // 1 GB
	cleanupTimeout = time.Hour
	cleanupPeriod  = 10 * time.Minute
)

type pendingUpload struct {
	dir       string
	chunks    int
	totalSize int64
	createdAt time.Time
	mu        sync.Mutex
}

// ChunkedUploadHandler handles chunked file uploads for large ZIP archives.
type ChunkedUploadHandler struct {
	uploads   sync.Map // map[string]*pendingUpload
	importSvc importService
}

// NewChunkedUploadHandler creates a new ChunkedUploadHandler.
func NewChunkedUploadHandler(importSvc importService) *ChunkedUploadHandler {
	return &ChunkedUploadHandler{importSvc: importSvc}
}

// StartCleanup starts a background goroutine that removes stale uploads older than 1 hour.
func (h *ChunkedUploadHandler) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(cleanupPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.runCleanup()
			}
		}
	}()
}

func (h *ChunkedUploadHandler) runCleanup() {
	now := time.Now()
	h.uploads.Range(func(key, value any) bool {
		pu := value.(*pendingUpload)
		pu.mu.Lock()
		old := now.Sub(pu.createdAt) > cleanupTimeout
		dir := pu.dir
		pu.mu.Unlock()
		if old {
			_ = os.RemoveAll(dir)
			h.uploads.Delete(key)
		}
		return true
	})
}

// InitUpload godoc
// @Summary      Initialise a chunked upload session
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/uploads/init [post]
func (h *ChunkedUploadHandler) InitUpload(c *gin.Context) {
	id := uuid.New().String()
	dir := filepath.Join(os.TempDir(), "chunked-"+id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}
	h.uploads.Store(id, &pendingUpload{
		dir:       dir,
		createdAt: time.Now(),
	})
	c.JSON(http.StatusOK, gin.H{"upload_id": id})
}

// UploadChunk godoc
// @Summary      Upload one chunk of a pending upload
// @Tags         admin
// @Accept       application/octet-stream
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        upload_id  path      string  true  "Upload ID"
// @Param        index      path      int     true  "Chunk index (0-based)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      413  {object}  map[string]string
// @Router       /admin/uploads/{upload_id}/chunks/{index} [post]
func (h *ChunkedUploadHandler) UploadChunk(c *gin.Context) {
	uploadID := c.Param("upload_id")
	indexStr := c.Param("index")

	idx, err := strconv.Atoi(indexStr)
	if err != nil || idx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index must be a non-negative integer"})
		return
	}

	val, ok := h.uploads.Load(uploadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		return
	}
	pu := val.(*pendingUpload)

	chunkPath := filepath.Join(pu.dir, fmt.Sprintf("chunk-%08d", idx))

	// Idempotency: return 409 if chunk already written.
	if _, err := os.Stat(chunkPath); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "chunk already received"})
		return
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read chunk body"})
		return
	}

	pu.mu.Lock()
	newTotal := pu.totalSize + int64(len(data))
	pu.mu.Unlock()

	if newTotal > maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "upload exceeds 1 GB limit"})
		return
	}

	if err := os.WriteFile(chunkPath, data, 0o600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write chunk"})
		return
	}

	pu.mu.Lock()
	pu.chunks++
	pu.totalSize += int64(len(data))
	pu.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"index":          idx,
		"received_bytes": len(data),
	})
}

// completeRequest is the JSON body for CompleteUpload.
type completeRequest struct {
	Name       string `json:"name"`
	EffectType string `json:"effect_type"`
}

// CompleteUpload godoc
// @Summary      Assemble chunks and import the ZIP archive
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        upload_id  path      string           true  "Upload ID"
// @Param        body       body      completeRequest  true  "Study metadata"
// @Success      201  {object}  service.ImportArchiveResult
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/uploads/{upload_id}/complete [post]
func (h *ChunkedUploadHandler) CompleteUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	val, ok := h.uploads.Load(uploadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		return
	}
	pu := val.(*pendingUpload)

	var req completeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.EffectType = strings.TrimSpace(req.EffectType)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.EffectType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "effect_type is required"})
		return
	}

	// Collect chunk files in order.
	entries, err := os.ReadDir(pu.dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk directory"})
		return
	}
	if len(entries) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no chunks received"})
		return
	}

	// Assemble chunks into a single temp ZIP file.
	assembled, err := os.CreateTemp("", "assembled-*.zip")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create assembled file"})
		return
	}
	assembledPath := assembled.Name()
	defer func() { _ = os.Remove(assembledPath) }()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		chunkPath := filepath.Join(pu.dir, entry.Name())
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			_ = assembled.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read chunk %s", entry.Name())})
			return
		}
		if _, err := assembled.Write(chunkData); err != nil {
			_ = assembled.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write assembled file"})
			return
		}
	}

	// Get assembled size before seeking.
	assembledSize, err := assembled.Seek(0, io.SeekCurrent)
	if err != nil {
		_ = assembled.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get assembled size"})
		return
	}

	// Seek back to start for reading.
	if _, err := assembled.Seek(0, io.SeekStart); err != nil {
		_ = assembled.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to seek assembled file"})
		return
	}

	// Clean up chunk directory now that assembly is done.
	_ = os.RemoveAll(pu.dir)
	h.uploads.Delete(uploadID)

	importReq := service.ImportArchiveRequest{
		Name:       req.Name,
		EffectType: req.EffectType,
		ZIPReader:  assembled,
		ZIPSize:    assembledSize,
	}

	result, err := h.importSvc.ImportArchive(c.Request.Context(), importReq)
	_ = assembled.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// AbortUpload godoc
// @Summary      Abort a pending chunked upload
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        upload_id  path      string  true  "Upload ID"
// @Success      200  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /admin/uploads/{upload_id} [delete]
func (h *ChunkedUploadHandler) AbortUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	val, ok := h.uploads.Load(uploadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload not found"})
		return
	}
	pu := val.(*pendingUpload)
	_ = os.RemoveAll(pu.dir)
	h.uploads.Delete(uploadID)

	c.JSON(http.StatusOK, gin.H{"message": "upload aborted"})
}

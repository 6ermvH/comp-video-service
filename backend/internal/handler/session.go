package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/service"
)

type sessionService interface {
	Start(ctx context.Context, req *model.StartSessionRequest) (*service.SessionStartResult, error)
	NextTask(ctx context.Context, token string) (*model.TaskPayload, error)
	Complete(ctx context.Context, token string) (*service.SessionCompleteResult, error)
}

// SessionHandler serves participant session endpoints.
type SessionHandler struct {
	sessionSvc sessionService
}

func NewSessionHandler(sessionSvc sessionService) *SessionHandler {
	return &SessionHandler{sessionSvc: sessionSvc}
}

// Start godoc
// @Summary      Start participant session
// @Tags         session
// @Accept       json
// @Produce      json
// @Param        body  body      model.StartSessionRequest  true  "Session data"
// @Success      200   {object}  service.SessionStartResult
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /session/start [post]
func (h *SessionHandler) Start(c *gin.Context) {
	var req model.StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.sessionSvc.Start(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// NextTask godoc
// @Summary      Get next task for session
// @Tags         session
// @Produce      json
// @Param        token  path      string            true  "Session token"
// @Success      200    {object}  model.TaskPayload
// @Success      204    "No more tasks"
// @Failure      404    {object}  map[string]string
// @Router       /session/{token}/next-task [get]
func (h *SessionHandler) NextTask(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session token is required"})
		return
	}

	task, err := h.sessionSvc.NextTask(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNoContent, gin.H{"message": "no more tasks"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, task)
}

// Complete godoc
// @Summary      Complete participant session
// @Tags         session
// @Produce      json
// @Param        token  path      string                         true  "Session token"
// @Success      200    {object}  service.SessionCompleteResult
// @Failure      404    {object}  map[string]string
// @Router       /session/{token}/complete [post]
func (h *SessionHandler) Complete(c *gin.Context) {
	result, err := h.sessionSvc.Complete(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

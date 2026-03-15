package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/service"
)

// SessionHandler serves participant session endpoints.
type SessionHandler struct {
	sessionSvc *service.SessionService
}

func NewSessionHandler(sessionSvc *service.SessionService) *SessionHandler {
	return &SessionHandler{sessionSvc: sessionSvc}
}

// Start godoc
// POST /api/session/start
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
// GET /api/session/:token/next-task
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
// POST /api/session/:token/complete
func (h *SessionHandler) Complete(c *gin.Context) {
	result, err := h.sessionSvc.Complete(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

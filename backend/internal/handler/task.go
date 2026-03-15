package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/service"
)

// TaskHandler handles task response and event endpoints.
type TaskHandler struct {
	sessionSvc      *service.SessionService
	pairRepo        *repository.PairPresentationRepository
	interactionRepo *repository.InteractionLogRepository
}

func NewTaskHandler(
	sessionSvc *service.SessionService,
	pairRepo *repository.PairPresentationRepository,
	interactionRepo *repository.InteractionLogRepository,
) *TaskHandler {
	return &TaskHandler{sessionSvc: sessionSvc, pairRepo: pairRepo, interactionRepo: interactionRepo}
}

// SubmitResponse godoc
// POST /api/task/:id/response
func (h *TaskHandler) SubmitResponse(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req model.TaskResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.sessionSvc.SaveResponse(c.Request.Context(), taskID, &req)
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "response already submitted"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// LogEvent godoc
// POST /api/task/:id/event
func (h *TaskHandler) LogEvent(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req model.TaskEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.EventType) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event_type is required"})
		return
	}

	pp, err := h.pairRepo.GetByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	event, err := h.interactionRepo.Create(c.Request.Context(), &model.InteractionLog{
		ParticipantID:      pp.ParticipantID,
		PairPresentationID: &pp.ID,
		EventType:          req.EventType,
		PayloadJSON:        req.PayloadJSON,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

func isUniqueViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "uq_response_presentation")
}

package handler

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/service"
)

type studyService interface {
	ListStudies(ctx context.Context) ([]*model.Study, error)
	CreateStudy(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Study, error)
	ListGroups(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error)
	CreateGroup(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error)
	ImportSourceItemsCSV(ctx context.Context, studyID uuid.UUID, r io.Reader) (int, error)
	ListSourceItems(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItem, error)
}

type assetService interface {
	Upload(ctx context.Context, input service.AssetUploadInput) (*model.Video, error)
}

type analyticsService interface {
	Overview(ctx context.Context) (*service.AnalyticsOverview, error)
	StudyDetail(ctx context.Context, studyID uuid.UUID) (*service.StudyAnalytics, error)
}

type qcService interface {
	BuildReport(ctx context.Context) (*service.QCReport, error)
}

type exportService interface {
	ExportCSV(ctx context.Context) ([]byte, error)
	ExportJSON(ctx context.Context) ([]byte, error)
}

// AdminHandler handles all admin-only endpoints.
type AdminHandler struct {
	studySvc     studyService
	assetSvc     assetService
	analyticsSvc analyticsService
	qcSvc        qcService
	exportSvc    exportService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	studySvc studyService,
	assetSvc assetService,
	analyticsSvc analyticsService,
	qcSvc qcService,
	exportSvc exportService,
) *AdminHandler {
	return &AdminHandler{
		studySvc:     studySvc,
		assetSvc:     assetSvc,
		analyticsSvc: analyticsSvc,
		qcSvc:        qcSvc,
		exportSvc:    exportSvc,
	}
}

// ListStudies godoc
// GET /api/admin/studies
func (h *AdminHandler) ListStudies(c *gin.Context) {
	studies, err := h.studySvc.ListStudies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if studies == nil {
		studies = make([]*model.Study, 0)
	}
	c.JSON(http.StatusOK, gin.H{"studies": studies})
}

// CreateStudy godoc
// POST /api/admin/studies
func (h *AdminHandler) CreateStudy(c *gin.Context) {
	var req model.CreateStudyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	study, err := h.studySvc.CreateStudy(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, study)
}

// PatchStudyStatus godoc
// PATCH /api/admin/studies/:id
func (h *AdminHandler) PatchStudyStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}
	var req model.UpdateStudyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	study, err := h.studySvc.UpdateStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, study)
}

// CreateGroup godoc
// POST /api/admin/studies/:id/groups
func (h *AdminHandler) CreateGroup(c *gin.Context) {
	studyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}

	var req model.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group, err := h.studySvc.CreateGroup(c.Request.Context(), studyID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, group)
}

// ListGroups godoc
// GET /api/admin/studies/:id/groups
func (h *AdminHandler) ListGroups(c *gin.Context) {
	studyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}

	groups, err := h.studySvc.ListGroups(c.Request.Context(), studyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// ImportSourceItems godoc
// POST /api/admin/studies/:id/import
func (h *AdminHandler) ImportSourceItems(c *gin.Context) {
	studyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() {
		_ = file.Close()
	}()

	created, err := h.studySvc.ImportSourceItemsCSV(c.Request.Context(), studyID, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"created": created})
}

// UploadAsset godoc
// POST /api/admin/assets/upload (multipart/form-data)
// Fields: file(required), method_type(required), source_item_id(optional), title(optional), description(optional)
func (h *AdminHandler) UploadAsset(c *gin.Context) {
	methodType := strings.TrimSpace(c.PostForm("method_type"))
	if methodType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "method_type is required"})
		return
	}

	var sourceItemID *uuid.UUID
	if raw := strings.TrimSpace(c.PostForm("source_item_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source_item_id"})
			return
		}
		sourceItemID = &parsed
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() {
		_ = file.Close()
	}()

	if err := validateMP4(header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	asset, err := h.assetSvc.Upload(c.Request.Context(), service.AssetUploadInput{
		SourceItemID: sourceItemID,
		MethodType:   methodType,
		Title:        strings.TrimSpace(c.PostForm("title")),
		Description:  strings.TrimSpace(c.PostForm("description")),
		ContentType:  "video/mp4",
		Filename:     header.Filename,
		Size:         header.Size,
		Reader:       file,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, asset)
}

// ListSourceItems godoc
// GET /api/admin/source-items
func (h *AdminHandler) ListSourceItems(c *gin.Context) {
	var studyID *uuid.UUID
	var groupID *uuid.UUID

	if sid := c.Query("study_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study_id"})
			return
		}
		studyID = &parsed
	}
	if gid := c.Query("group_id"); gid != "" {
		parsed, err := uuid.Parse(gid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
			return
		}
		groupID = &parsed
	}

	items, err := h.studySvc.ListSourceItems(c.Request.Context(), studyID, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"source_items": items})
}

// AnalyticsOverview godoc
// GET /api/admin/analytics/overview
func (h *AdminHandler) AnalyticsOverview(c *gin.Context) {
	overview, err := h.analyticsSvc.Overview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

// AnalyticsStudy godoc
// GET /api/admin/analytics/study/:id
func (h *AdminHandler) AnalyticsStudy(c *gin.Context) {
	studyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}
	detail, err := h.analyticsSvc.StudyDetail(c.Request.Context(), studyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// AnalyticsQC godoc
// GET /api/admin/analytics/qc
func (h *AdminHandler) AnalyticsQC(c *gin.Context) {
	report, err := h.qcSvc.BuildReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// ExportCSV godoc
// GET /api/admin/export/csv
func (h *AdminHandler) ExportCSV(c *gin.Context) {
	payload, err := h.exportSvc.ExportCSV(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="responses.csv"`)
	c.Data(http.StatusOK, "text/csv", payload)
}

// ExportJSON godoc
// GET /api/admin/export/json
func (h *AdminHandler) ExportJSON(c *gin.Context) {
	payload, err := h.exportSvc.ExportJSON(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", payload)
}

func validateMP4(header *multipart.FileHeader) error {
	ct := header.Header.Get("Content-Type")
	name := strings.ToLower(header.Filename)
	if ct != "video/mp4" && !strings.HasSuffix(name, ".mp4") {
		return fmt.Errorf("only MP4 files are allowed (got %q)", ct)
	}
	return nil
}

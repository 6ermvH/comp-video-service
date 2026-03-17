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
	UpdateStudy(ctx context.Context, id uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error)
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
	PairBreakdown(ctx context.Context, studyID uuid.UUID) ([]*service.PairStat, error)
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
// @Summary      List all studies
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Success      200  {object}  map[string]interface{}
// @Router       /admin/studies [get]
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
// @Summary      Create a new study
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        body  body      model.CreateStudyRequest  true  "Study data"
// @Success      201   {object}  model.Study
// @Failure      400   {object}  map[string]string
// @Router       /admin/studies [post]
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

// UpdateStudy godoc
// @Summary      Update study
// @Description  Update study status and/or fields
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id    path      string                    true  "Study ID"
// @Param        body  body      model.UpdateStudyRequest  true  "Update payload"
// @Success      200   {object}  model.Study
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /admin/studies/{id} [patch]
func (h *AdminHandler) UpdateStudy(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}
	var req model.UpdateStudyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	study, err := h.studySvc.UpdateStudy(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, study)
}

// CreateGroup godoc
// @Summary      Create a group within a study
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id    path      string                    true  "Study ID"
// @Param        body  body      model.CreateGroupRequest  true  "Group data"
// @Success      201   {object}  model.Group
// @Failure      400   {object}  map[string]string
// @Router       /admin/studies/{id}/groups [post]
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
// @Summary      List groups for a study
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id  path      string  true  "Study ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /admin/studies/{id}/groups [get]
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
// @Summary      Import source items from CSV
// @Tags         admin
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id    path      string  true  "Study ID"
// @Param        file  formData  file    true  "CSV file"
// @Success      200   {object}  map[string]int
// @Failure      400   {object}  map[string]string
// @Router       /admin/studies/{id}/import [post]
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
// @Summary      Upload a video asset
// @Tags         admin
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        file            formData  file    true   "MP4 video file"
// @Param        method_type     formData  string  true   "baseline or candidate"
// @Param        source_item_id  formData  string  false  "Source item UUID to link"
// @Param        title           formData  string  false  "Video title"
// @Param        description     formData  string  false  "Video description"
// @Success      201  {object}   model.Video
// @Failure      400  {object}   map[string]string
// @Router       /admin/assets/upload [post]
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
// @Summary      List source items (pairs)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        study_id  query     string  false  "Filter by study UUID"
// @Param        group_id  query     string  false  "Filter by group UUID"
// @Success      200       {object}  map[string]interface{}
// @Router       /admin/source-items [get]
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
// @Summary      Get analytics overview
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Success      200  {object}  service.AnalyticsOverview
// @Router       /admin/analytics/overview [get]
func (h *AdminHandler) AnalyticsOverview(c *gin.Context) {
	overview, err := h.analyticsSvc.Overview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

// AnalyticsStudy godoc
// @Summary      Get per-study analytics
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id  path      string  true  "Study ID"
// @Success      200  {object}  service.StudyAnalytics
// @Router       /admin/analytics/study/{id} [get]
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

// AnalyticsPairs godoc
// GET /api/admin/analytics/study/:id/pairs
func (h *AdminHandler) AnalyticsPairs(c *gin.Context) {
	studyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}
	pairs, err := h.analyticsSvc.PairBreakdown(c.Request.Context(), studyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pairs)
}

// AnalyticsQC godoc
// @Summary      Get QC report
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Success      200  {object}  service.QCReport
// @Router       /admin/analytics/qc [get]
func (h *AdminHandler) AnalyticsQC(c *gin.Context) {
	report, err := h.qcSvc.BuildReport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// ExportCSV godoc
// @Summary      Export responses as CSV
// @Tags         admin
// @Produce      text/csv
// @Security     BearerAuth
// @Security     CSRFToken
// @Success      200  {string}  string
// @Router       /admin/export/csv [get]
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
// @Summary      Export responses as JSON
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Success      200  {array}   map[string]string
// @Router       /admin/export/json [get]
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

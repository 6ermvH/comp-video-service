package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
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
	ListSourceItems(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItemDetail, error)
	ListAssets(ctx context.Context, page, perPage int, search string) ([]*model.Video, int, error)
	ListFreeAssets(ctx context.Context) ([]*model.Video, error)
	CreatePair(ctx context.Context, studyID uuid.UUID, req *model.CreatePairRequest) (*model.SourceItem, error)
	DeletePair(ctx context.Context, id uuid.UUID) error
}

type importService interface {
	ImportArchive(ctx context.Context, req service.ImportArchiveRequest) (*service.ImportArchiveResult, error)
}

type assetService interface {
	Upload(ctx context.Context, input service.AssetUploadInput) (*model.Video, error)
	DeleteAsset(ctx context.Context, id uuid.UUID) error
	GetPresignedURL(ctx context.Context, id uuid.UUID) (string, error)
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
	importSvc    importService
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

// NewAdminHandlerWithImport creates a new AdminHandler with import support.
func NewAdminHandlerWithImport(
	studySvc studyService,
	assetSvc assetService,
	analyticsSvc analyticsService,
	qcSvc qcService,
	exportSvc exportService,
	importSvc importService,
) *AdminHandler {
	return &AdminHandler{
		studySvc:     studySvc,
		assetSvc:     assetSvc,
		analyticsSvc: analyticsSvc,
		qcSvc:        qcSvc,
		exportSvc:    exportSvc,
		importSvc:    importSvc,
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
// @Summary      List source items (pairs) with enriched data
// @Description  Returns pairs with group_name, asset_count and response_count.
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

// ListAssets godoc
// @Summary      List video assets (paginated)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page      query  int     false  "Page number (default 1)"
// @Param        per_page  query  int     false  "Items per page (default 20, max 100)"
// @Param        search    query  string  false  "Filter by title (case-insensitive substring)"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /admin/assets [get]
func (h *AdminHandler) ListAssets(c *gin.Context) {
	page := 1
	perPage := 20
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("per_page")); err == nil && v > 0 && v <= 100 {
		perPage = v
	}
	search := strings.TrimSpace(c.Query("search"))

	assets, total, err := h.studySvc.ListAssets(c.Request.Context(), page, perPage, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"assets":   assets,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// ListFreeAssets godoc
// @Summary      List free (unlinked) video assets
// @Description  Returns all video assets not linked to any pair, for use in pair builder.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /admin/assets/free [get]
func (h *AdminHandler) ListFreeAssets(c *gin.Context) {
	assets, err := h.studySvc.ListFreeAssets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"assets": assets})
}

// CreatePair godoc
// @Summary      Create a source item pair from two existing video assets
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id    path      string                   true  "Study ID"
// @Param        body  body      model.CreatePairRequest  true  "Pair payload"
// @Success      201   {object}  model.SourceItem
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /admin/studies/{id}/pairs [post]
func (h *AdminHandler) CreatePair(c *gin.Context) {
	studyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid study id"})
		return
	}
	var req model.CreatePairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.studySvc.CreatePair(c.Request.Context(), studyID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// DeletePair godoc
// @Summary      Delete a source item (pair)
// @Description  Deletes a pair and frees its video assets back to the library. Blocked if responses exist.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id   path      string  true  "Source item UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/source-items/{id} [delete]
func (h *AdminHandler) DeletePair(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.studySvc.DeletePair(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrPairHasResponses) {
			c.JSON(http.StatusConflict, gin.H{"error": "pair has participant responses and cannot be deleted"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// DeleteAsset godoc
// @Summary      Delete a video asset
// @Description  Deletes a free video asset. Blocked if still linked to a pair or used in presentations.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id   path      string  true  "Asset UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/assets/{id} [delete]
func (h *AdminHandler) DeleteAsset(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.assetSvc.DeleteAsset(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrAssetInUse) {
			c.JSON(http.StatusConflict, gin.H{"error": "asset is linked to a pair or in use — delete the pair first"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetAssetURL godoc
// @Summary      Get presigned URL for a video asset
// @Description  Returns a presigned (or public) URL to stream or download the video.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        id   path      string  true  "Asset UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/assets/{id}/url [get]
func (h *AdminHandler) GetAssetURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	url, err := h.assetSvc.GetPresignedURL(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAssetNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
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
// @Summary      Get per-pair analytics for a study
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Study ID"
// @Success      200  {array}   service.PairStat
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/analytics/study/{id}/pairs [get]
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

// ImportArchive godoc
// @Summary      Import a study from a ZIP archive of MP4 files
// @Description  Accepts a ZIP archive containing MP4 files named <group>_<name>_<baseline|candidate>.mp4.
// @Tags         admin
// @Accept       mpfd
// @Produce      json
// @Security     BearerAuth
// @Security     CSRFToken
// @Param        file                    formData  file    true   "ZIP archive"
// @Param        name                    formData  string  true   "Study name"
// @Param        effect_type             formData  string  true   "Effect type (flooding, explosion, mixed)"
// @Param        max_tasks_per_participant formData int    false  "Max tasks per participant"
// @Param        tie_option_enabled      formData  bool    false  "Enable tie option"
// @Param        reasons_enabled         formData  bool    false  "Enable reasons"
// @Param        confidence_enabled      formData  bool    false  "Enable confidence"
// @Success      201  {object}  service.ImportArchiveResult
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /admin/studies/import-archive [post]
func (h *AdminHandler) ImportArchive(c *gin.Context) {
	if h.importSvc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "import service not configured"})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	effectType := strings.TrimSpace(c.PostForm("effect_type"))
	if effectType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "effect_type is required"})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	zipData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read archive"})
		return
	}

	req := service.ImportArchiveRequest{
		Name:       name,
		EffectType: effectType,
		ZIPData:    zipData,
	}

	if v := strings.TrimSpace(c.PostForm("max_tasks_per_participant")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "max_tasks_per_participant must be an integer"})
			return
		}
		req.MaxTasksPerParticipant = n
	}

	if v := strings.TrimSpace(c.PostForm("tie_option_enabled")); v != "" {
		b := v == "true" || v == "1"
		req.TieOptionEnabled = &b
	}
	if v := strings.TrimSpace(c.PostForm("reasons_enabled")); v != "" {
		b := v == "true" || v == "1"
		req.ReasonsEnabled = &b
	}
	if v := strings.TrimSpace(c.PostForm("confidence_enabled")); v != "" {
		b := v == "true" || v == "1"
		req.ConfidenceEnabled = &b
	}

	result, err := h.importSvc.ImportArchive(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func validateMP4(header *multipart.FileHeader) error {
	ct := header.Header.Get("Content-Type")
	name := strings.ToLower(header.Filename)
	if ct != "video/mp4" && !strings.HasSuffix(name, ".mp4") {
		return fmt.Errorf("only MP4 files are allowed (got %q)", ct)
	}
	return nil
}

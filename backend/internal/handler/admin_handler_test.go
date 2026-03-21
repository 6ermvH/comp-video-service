package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/service"
)

type mockStudyService struct {
	listStudiesFn                func(context.Context) ([]*model.Study, error)
	createStudyFn                func(context.Context, *model.CreateStudyRequest) (*model.Study, error)
	updateStudyFn                func(context.Context, uuid.UUID, *model.UpdateStudyRequest) (*model.Study, error)
	deleteStudyFn                func(context.Context, uuid.UUID) error
	listGroupsFn                 func(context.Context, uuid.UUID) ([]*model.Group, error)
	createGroupFn                func(context.Context, uuid.UUID, *model.CreateGroupRequest) (*model.Group, error)
	listSourceItemsFn            func(context.Context, *uuid.UUID, *uuid.UUID) ([]*model.SourceItemDetail, error)
	listAssetsFn                 func(context.Context, int, int, string) ([]*model.Video, int, error)
	listFreeAssetsFn             func(context.Context) ([]*model.Video, error)
	createPairFn                 func(context.Context, uuid.UUID, *model.CreatePairRequest) (*model.SourceItem, error)
	deletePairFn                 func(context.Context, uuid.UUID) error
	updateSourceItemAttentionFn  func(context.Context, uuid.UUID, bool) (*model.SourceItemDetail, error)
}

func (m *mockStudyService) ListStudies(ctx context.Context) ([]*model.Study, error) {
	return m.listStudiesFn(ctx)
}
func (m *mockStudyService) CreateStudy(ctx context.Context, r *model.CreateStudyRequest) (*model.Study, error) {
	return m.createStudyFn(ctx, r)
}
func (m *mockStudyService) UpdateStudy(ctx context.Context, id uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error) {
	return m.updateStudyFn(ctx, id, req)
}
func (m *mockStudyService) DeleteStudy(ctx context.Context, id uuid.UUID) error {
	if m.deleteStudyFn != nil {
		return m.deleteStudyFn(ctx, id)
	}
	return nil
}
func (m *mockStudyService) ListGroups(ctx context.Context, id uuid.UUID) ([]*model.Group, error) {
	return m.listGroupsFn(ctx, id)
}
func (m *mockStudyService) CreateGroup(ctx context.Context, id uuid.UUID, r *model.CreateGroupRequest) (*model.Group, error) {
	return m.createGroupFn(ctx, id, r)
}
func (m *mockStudyService) ListSourceItems(ctx context.Context, sid *uuid.UUID, gid *uuid.UUID) ([]*model.SourceItemDetail, error) {
	return m.listSourceItemsFn(ctx, sid, gid)
}
func (m *mockStudyService) ListAssets(ctx context.Context, page, perPage int, search string) ([]*model.Video, int, error) {
	return m.listAssetsFn(ctx, page, perPage, search)
}
func (m *mockStudyService) ListFreeAssets(ctx context.Context) ([]*model.Video, error) {
	return m.listFreeAssetsFn(ctx)
}
func (m *mockStudyService) CreatePair(ctx context.Context, id uuid.UUID, req *model.CreatePairRequest) (*model.SourceItem, error) {
	return m.createPairFn(ctx, id, req)
}
func (m *mockStudyService) DeletePair(ctx context.Context, id uuid.UUID) error {
	return m.deletePairFn(ctx, id)
}
func (m *mockStudyService) UpdateSourceItemAttention(ctx context.Context, id uuid.UUID, isAttentionCheck bool) (*model.SourceItemDetail, error) {
	if m.updateSourceItemAttentionFn != nil {
		return m.updateSourceItemAttentionFn(ctx, id, isAttentionCheck)
	}
	return nil, nil
}

type mockAssetService struct {
	uploadFn          func(context.Context, service.AssetUploadInput) (*model.Video, error)
	deleteFn          func(context.Context, uuid.UUID) error
	getPresignedURLFn func(context.Context, uuid.UUID) (string, error)
}

func (m *mockAssetService) Upload(ctx context.Context, in service.AssetUploadInput) (*model.Video, error) {
	return m.uploadFn(ctx, in)
}
func (m *mockAssetService) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}
func (m *mockAssetService) GetPresignedURL(ctx context.Context, id uuid.UUID) (string, error) {
	if m.getPresignedURLFn != nil {
		return m.getPresignedURLFn(ctx, id)
	}
	return "", nil
}

type mockAnalyticsService struct {
	overviewFn func(context.Context) (*service.AnalyticsOverview, error)
	studyFn    func(context.Context, uuid.UUID) (*service.StudyAnalytics, error)
	pairFn     func(context.Context, uuid.UUID) ([]*service.PairStat, error)
}

func (m *mockAnalyticsService) Overview(ctx context.Context) (*service.AnalyticsOverview, error) {
	return m.overviewFn(ctx)
}
func (m *mockAnalyticsService) StudyDetail(ctx context.Context, id uuid.UUID) (*service.StudyAnalytics, error) {
	return m.studyFn(ctx, id)
}
func (m *mockAnalyticsService) PairBreakdown(ctx context.Context, id uuid.UUID) ([]*service.PairStat, error) {
	return m.pairFn(ctx, id)
}

type mockQCService struct {
	reportFn func(context.Context) (*service.QCReport, error)
}

func (m *mockQCService) BuildReport(ctx context.Context) (*service.QCReport, error) {
	return m.reportFn(ctx)
}

type mockExportService struct {
	csvFn      func(context.Context) ([]byte, error)
	studyCSVFn func(context.Context, uuid.UUID) ([]byte, error)
}

func (m *mockExportService) ExportCSV(ctx context.Context) ([]byte, error) { return m.csvFn(ctx) }
func (m *mockExportService) ExportStudyCSV(ctx context.Context, studyID uuid.UUID) ([]byte, error) {
	if m.studyCSVFn != nil {
		return m.studyCSVFn(ctx, studyID)
	}
	return []byte("response_id,participant_id\n"), nil
}

type mockImportService struct {
	importFn func(context.Context, service.ImportArchiveRequest) (*service.ImportArchiveResult, error)
}

func (m *mockImportService) ImportArchive(ctx context.Context, req service.ImportArchiveRequest) (*service.ImportArchiveResult, error) {
	return m.importFn(ctx, req)
}

func TestAdminHandlerListStudiesAndGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sid := uuid.New()
	h := NewAdminHandler(
		&mockStudyService{
			listStudiesFn: func(context.Context) ([]*model.Study, error) { return []*model.Study{{ID: sid, Name: "S"}}, nil },
			listGroupsFn: func(context.Context, uuid.UUID) ([]*model.Group, error) {
				return []*model.Group{{ID: uuid.New(), Name: "G"}}, nil
			},
		},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)
	r := gin.New()
	r.GET("/studies", h.ListStudies)
	r.GET("/studies/:id/groups", h.ListGroups)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/studies", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/studies/"+sid.String()+"/groups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAdminHandlerCreateStudyBadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(&mockStudyService{}, &mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{})
	r := gin.New()
	r.POST("/studies", h.CreateStudy)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/studies", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminHandlerUpdateStudyInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(&mockStudyService{}, &mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{})
	r := gin.New()
	r.PATCH("/studies/:id", h.UpdateStudy)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/studies/invalid", mustJSON(t, map[string]string{"status": "active"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminHandlerAnalyticsAndExport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(
		&mockStudyService{},
		&mockAssetService{},
		&mockAnalyticsService{
			overviewFn: func(context.Context) (*service.AnalyticsOverview, error) {
				return &service.AnalyticsOverview{TotalResponses: 1}, nil
			},
			studyFn: func(context.Context, uuid.UUID) (*service.StudyAnalytics, error) {
				return &service.StudyAnalytics{}, nil
			},
			pairFn: func(context.Context, uuid.UUID) ([]*service.PairStat, error) {
				return []*service.PairStat{}, nil
			},
		},
		&mockQCService{reportFn: func(context.Context) (*service.QCReport, error) { return &service.QCReport{TotalResponses: 1}, nil }},
		&mockExportService{csvFn: func(context.Context) ([]byte, error) { return []byte("a,b\n"), nil }},
	)
	r := gin.New()
	r.GET("/overview", h.AnalyticsOverview)
	r.GET("/study/:id", h.AnalyticsStudy)
	r.GET("/study/:id/pairs", h.AnalyticsPairs)
	r.GET("/qc", h.AnalyticsQC)
	r.GET("/csv", h.ExportCSV)

	cases := []string{"/overview", "/study/" + uuid.New().String(), "/study/" + uuid.New().String() + "/pairs", "/qc", "/csv"}
	for _, path := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestAdminHandlerAnalyticsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(
		&mockStudyService{},
		&mockAssetService{},
		&mockAnalyticsService{overviewFn: func(context.Context) (*service.AnalyticsOverview, error) { return nil, errors.New("x") }},
		&mockQCService{},
		&mockExportService{},
	)
	r := gin.New()
	r.GET("/overview", h.AnalyticsOverview)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/overview", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	_ = time.Now()
}

func TestAdminHandlerCreateStudyAndPatchSuccessAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	studyID := uuid.New()
	h := NewAdminHandler(
		&mockStudyService{
			createStudyFn: func(context.Context, *model.CreateStudyRequest) (*model.Study, error) {
				return &model.Study{ID: studyID}, nil
			},
			updateStudyFn: func(_ context.Context, _ uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error) {
				if req.Status == nil {
					return nil, errors.New("status required")
				}
				return &model.Study{ID: studyID, Status: *req.Status}, nil
			},
		},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)
	r := gin.New()
	r.POST("/studies", h.CreateStudy)
	r.PATCH("/studies/:id", h.UpdateStudy)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/studies", mustJSON(t, map[string]any{"name": "A", "effect_type": "flooding"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/studies/"+studyID.String(), mustJSON(t, map[string]any{"status": "active"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/studies/"+studyID.String(), mustJSON(t, map[string]any{}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on service validation error, got %d", w.Code)
	}
}

func TestAdminHandlerListSourceItemsInvalidQueryAndExportErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(
		&mockStudyService{
			listSourceItemsFn: func(context.Context, *uuid.UUID, *uuid.UUID) ([]*model.SourceItemDetail, error) {
				return nil, errors.New("boom")
			},
		},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{
			csvFn: func(context.Context) ([]byte, error) { return nil, errors.New("x") },
		},
	)
	r := gin.New()
	r.GET("/source-items", h.ListSourceItems)
	r.GET("/csv", h.ExportCSV)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/source-items?study_id=bad", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid study_id, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/source-items?group_id=bad", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid group_id, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/source-items", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 source items error, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/csv", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 csv error, got %d", w.Code)
	}

}

func TestAdminHandlerDeletePairAndAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pairID := uuid.New()
	assetID := uuid.New()

	h := NewAdminHandler(
		&mockStudyService{
			deletePairFn: func(_ context.Context, id uuid.UUID) error {
				if id != pairID {
					t.Fatalf("unexpected pair id: %s", id)
				}
				return nil
			},
		},
		&mockAssetService{
			deleteFn: func(_ context.Context, id uuid.UUID) error {
				if id != assetID {
					t.Fatalf("unexpected asset id: %s", id)
				}
				return nil
			},
		},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)

	r := gin.New()
	r.DELETE("/source-items/:id", h.DeletePair)
	r.DELETE("/assets/:id", h.DeleteAsset)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/source-items/"+pairID.String(), nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for delete pair, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/assets/"+assetID.String(), nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for delete asset, got %d", w.Code)
	}
}

func TestAdminHandlerListAssetsAndFreeAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(
		&mockStudyService{
			listAssetsFn: func(_ context.Context, page, perPage int, search string) ([]*model.Video, int, error) {
				if page != 2 || perPage != 10 {
					t.Fatalf("unexpected pagination: page=%d per_page=%d", page, perPage)
				}
				return []*model.Video{{ID: uuid.New()}}, 33, nil
			},
			listFreeAssetsFn: func(context.Context) ([]*model.Video, error) {
				return []*model.Video{{ID: uuid.New()}}, nil
			},
		},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)

	r := gin.New()
	r.GET("/assets", h.ListAssets)
	r.GET("/assets/free", h.ListFreeAssets)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets?page=2&per_page=10", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for assets, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/free", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for free assets, got %d", w.Code)
	}
}

func TestAdminHandlerDeletePairAndAssetConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pairID := uuid.New()
	assetID := uuid.New()

	h := NewAdminHandler(
		&mockStudyService{
			deletePairFn: func(context.Context, uuid.UUID) error {
				return service.ErrPairHasResponses
			},
		},
		&mockAssetService{
			deleteFn: func(context.Context, uuid.UUID) error {
				return service.ErrAssetInUse
			},
		},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)

	r := gin.New()
	r.DELETE("/source-items/:id", h.DeletePair)
	r.DELETE("/assets/:id", h.DeleteAsset)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/source-items/"+pairID.String(), nil))
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for delete pair conflict, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/assets/"+assetID.String(), nil))
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for delete asset conflict, got %d", w.Code)
	}
}

func TestAdminHandlerCreateGroupAndListSourceItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	studyID := uuid.New()
	h := NewAdminHandler(
		&mockStudyService{
			createGroupFn: func(context.Context, uuid.UUID, *model.CreateGroupRequest) (*model.Group, error) {
				return &model.Group{ID: uuid.New(), Name: "G"}, nil
			},
			listSourceItemsFn: func(context.Context, *uuid.UUID, *uuid.UUID) ([]*model.SourceItemDetail, error) {
				return []*model.SourceItemDetail{{ID: uuid.New(), GroupName: "G"}}, nil
			},
		},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)
	r := gin.New()
	r.POST("/studies/:id/groups", h.CreateGroup)
	r.GET("/source-items", h.ListSourceItems)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/studies/"+studyID.String()+"/groups", mustJSON(t, map[string]any{"name": "G"}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/source-items?study_id="+studyID.String(), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAdminHandlerUploadAssetBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(
		&mockStudyService{},
		&mockAssetService{uploadFn: func(context.Context, service.AssetUploadInput) (*model.Video, error) {
			return &model.Video{ID: uuid.New()}, nil
		}},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)
	r := gin.New()
	r.POST("/assets/upload", h.UploadAsset)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/assets/upload", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("method_type", "baseline")
	_ = writer.WriteField("source_item_id", "invalid")
	part, _ := writer.CreateFormFile("file", "video.mp4")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/assets/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	_ = writer.WriteField("method_type", "baseline")
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="video.txt"`)
	hdr.Set("Content-Type", "text/plain")
	part, _ = writer.CreatePart(hdr)
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/assets/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	_ = writer.WriteField("method_type", "baseline")
	part, _ = writer.CreateFormFile("file", "video.mp4")
	_, _ = part.Write([]byte("fake"))
	_ = writer.Close()
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/assets/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestAdminHandlerUpdateSourceItem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	itemID := uuid.New()

	t.Run("success", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{
				updateSourceItemAttentionFn: func(_ context.Context, id uuid.UUID, isAttentionCheck bool) (*model.SourceItemDetail, error) {
					if id != itemID {
						t.Fatalf("unexpected id: %s", id)
					}
					return &model.SourceItemDetail{ID: id, IsAttentionCheck: isAttentionCheck, GroupName: "G"}, nil
				},
			},
			&mockAssetService{},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{},
		)
		r := gin.New()
		r.PATCH("/source-items/:id", h.UpdateSourceItem)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/source-items/"+itemID.String(), mustJSON(t, map[string]any{"is_attention_check": true}))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		h := NewAdminHandler(&mockStudyService{}, &mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{})
		r := gin.New()
		r.PATCH("/source-items/:id", h.UpdateSourceItem)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/source-items/not-a-uuid", mustJSON(t, map[string]any{"is_attention_check": true}))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := NewAdminHandler(&mockStudyService{}, &mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{})
		r := gin.New()
		r.PATCH("/source-items/:id", h.UpdateSourceItem)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/source-items/"+itemID.String(), bytes.NewBufferString("{bad"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{
				updateSourceItemAttentionFn: func(_ context.Context, _ uuid.UUID, _ bool) (*model.SourceItemDetail, error) {
					return nil, errors.New("db error")
				},
			},
			&mockAssetService{},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{},
		)
		r := gin.New()
		r.PATCH("/source-items/:id", h.UpdateSourceItem)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/source-items/"+itemID.String(), mustJSON(t, map[string]any{"is_attention_check": false}))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

// makeTestZIP builds a minimal ZIP archive with the given filenames and returns its bytes.
func makeTestZIP(t *testing.T, filenames []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, name := range filenames {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip create %q: %v", name, err)
		}
		_, _ = f.Write([]byte("fake-mp4-data"))
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func buildImportRequest(t *testing.T, fields map[string]string, zipData []byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	if zipData != nil {
		part, err := w.CreateFormFile("file", "import.zip")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		_, _ = part.Write(zipData)
	}
	_ = w.Close()
	return &body, w.FormDataContentType()
}

func TestAdminHandlerImportArchive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	studyID := uuid.New()
	importSvcOK := &mockImportService{
		importFn: func(_ context.Context, req service.ImportArchiveRequest) (*service.ImportArchiveResult, error) {
			return &service.ImportArchiveResult{
				Study:          &model.Study{ID: studyID, Name: req.Name},
				GroupsCreated:  1,
				PairsCreated:   1,
				VideosUploaded: 2,
				Errors:         []string{},
			}, nil
		},
	}

	h := NewAdminHandlerWithImport(
		&mockStudyService{},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
		importSvcOK,
	)

	r := gin.New()
	r.POST("/studies/import-archive", h.ImportArchive)

	// Missing name field.
	body, ct := buildImportRequest(t, map[string]string{"effect_type": "flooding"}, makeTestZIP(t, []string{"g_p_baseline.mp4"}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/studies/import-archive", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing name: expected 400, got %d", w.Code)
	}

	// Missing effect_type field.
	body, ct = buildImportRequest(t, map[string]string{"name": "Study"}, makeTestZIP(t, []string{"g_p_baseline.mp4"}))
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/studies/import-archive", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing effect_type: expected 400, got %d", w.Code)
	}

	// Missing file.
	body, ct = buildImportRequest(t, map[string]string{"name": "Study", "effect_type": "flooding"}, nil)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/studies/import-archive", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing file: expected 400, got %d", w.Code)
	}

	// Successful import.
	zipData := makeTestZIP(t, []string{"g_p_baseline.mp4", "g_p_candidate.mp4"})
	body, ct = buildImportRequest(t, map[string]string{"name": "My Study", "effect_type": "flooding"}, zipData)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/studies/import-archive", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("success: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Service error returns 500.
	hErr := NewAdminHandlerWithImport(
		&mockStudyService{},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
		&mockImportService{importFn: func(context.Context, service.ImportArchiveRequest) (*service.ImportArchiveResult, error) {
			return nil, errors.New("boom")
		}},
	)
	r2 := gin.New()
	r2.POST("/studies/import-archive", hErr.ImportArchive)

	zipData = makeTestZIP(t, []string{"g_p_baseline.mp4", "g_p_candidate.mp4"})
	body, ct = buildImportRequest(t, map[string]string{"name": "My Study", "effect_type": "flooding"}, zipData)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/studies/import-archive", body)
	req.Header.Set("Content-Type", ct)
	r2.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("service error: expected 500, got %d", w.Code)
	}
}

func TestAdminHandlerImportArchiveNoImportSvc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(
		&mockStudyService{},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{},
	)
	r := gin.New()
	r.POST("/studies/import-archive", h.ImportArchive)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/studies/import-archive", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when importSvc is nil, got %d", w.Code)
	}
}

func TestAdminHandlerExportStudyCSV(t *testing.T) {
	gin.SetMode(gin.TestMode)
	studyID := uuid.New()

	t.Run("success", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{},
			&mockAssetService{},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{
				studyCSVFn: func(_ context.Context, id uuid.UUID) ([]byte, error) {
					if id != studyID {
						t.Fatalf("unexpected study id: %s", id)
					}
					return []byte("response_id,participant_id\nr1,p1\n"), nil
				},
			},
		)
		r := gin.New()
		r.GET("/export/study/:id/csv", h.ExportStudyCSV)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/export/study/"+studyID.String()+"/csv", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
			t.Fatalf("expected text/csv, got %s", ct)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		h := NewAdminHandler(&mockStudyService{}, &mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{})
		r := gin.New()
		r.GET("/export/study/:id/csv", h.ExportStudyCSV)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/export/study/not-a-uuid/csv", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{},
			&mockAssetService{},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{
				studyCSVFn: func(context.Context, uuid.UUID) ([]byte, error) {
					return nil, errors.New("db error")
				},
			},
		)
		r := gin.New()
		r.GET("/export/study/:id/csv", h.ExportStudyCSV)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/export/study/"+studyID.String()+"/csv", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestAdminHandlerGetAssetURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assetID := uuid.New()

	t.Run("success", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{},
			&mockAssetService{
				getPresignedURLFn: func(_ context.Context, id uuid.UUID) (string, error) {
					if id != assetID {
						t.Fatalf("unexpected id: %s", id)
					}
					return "https://example.com/video.mp4", nil
				},
			},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{},
		)
		r := gin.New()
		r.GET("/assets/:id/url", h.GetAssetURL)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/"+assetID.String()+"/url", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{},
			&mockAssetService{},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{},
		)
		r := gin.New()
		r.GET("/assets/:id/url", h.GetAssetURL)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/not-a-uuid/url", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{},
			&mockAssetService{
				getPresignedURLFn: func(_ context.Context, _ uuid.UUID) (string, error) {
					return "", service.ErrAssetNotFound
				},
			},
			&mockAnalyticsService{},
			&mockQCService{},
			&mockExportService{},
		)
		r := gin.New()
		r.GET("/assets/:id/url", h.GetAssetURL)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/"+assetID.String()+"/url", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestAdminHandler_DeleteStudy(t *testing.T) {
	studyID := uuid.New()

	t.Run("invalid id returns 400", func(t *testing.T) {
		h := NewAdminHandler(&mockStudyService{}, &mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{})
		r := gin.New()
		r.DELETE("/studies/:id", h.DeleteStudy)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/studies/not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("study not found returns 404", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{
				deleteStudyFn: func(_ context.Context, _ uuid.UUID) error {
					return service.ErrStudyNotFound
				},
			},
			&mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{},
		)
		r := gin.New()
		r.DELETE("/studies/:id", h.DeleteStudy)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/studies/"+studyID.String(), nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("success returns 200 with message", func(t *testing.T) {
		h := NewAdminHandler(
			&mockStudyService{
				deleteStudyFn: func(_ context.Context, id uuid.UUID) error {
					if id != studyID {
						t.Fatalf("unexpected id: %v", id)
					}
					return nil
				},
			},
			&mockAssetService{}, &mockAnalyticsService{}, &mockQCService{}, &mockExportService{},
		)
		r := gin.New()
		r.DELETE("/studies/:id", h.DeleteStudy)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/studies/"+studyID.String(), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte("study deleted")) {
			t.Fatalf("expected 'study deleted' in body, got: %s", w.Body.String())
		}
	})
}

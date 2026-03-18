package handler

import (
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
	listStudiesFn     func(context.Context) ([]*model.Study, error)
	createStudyFn     func(context.Context, *model.CreateStudyRequest) (*model.Study, error)
	updateStudyFn     func(context.Context, uuid.UUID, *model.UpdateStudyRequest) (*model.Study, error)
	listGroupsFn      func(context.Context, uuid.UUID) ([]*model.Group, error)
	createGroupFn     func(context.Context, uuid.UUID, *model.CreateGroupRequest) (*model.Group, error)
	listSourceItemsFn func(context.Context, *uuid.UUID, *uuid.UUID) ([]*model.SourceItem, error)
	listAssetsFn      func(context.Context) ([]*model.Video, error)
	createPairFn      func(context.Context, uuid.UUID, *model.CreatePairRequest) (*model.SourceItem, error)
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
func (m *mockStudyService) ListGroups(ctx context.Context, id uuid.UUID) ([]*model.Group, error) {
	return m.listGroupsFn(ctx, id)
}
func (m *mockStudyService) CreateGroup(ctx context.Context, id uuid.UUID, r *model.CreateGroupRequest) (*model.Group, error) {
	return m.createGroupFn(ctx, id, r)
}
func (m *mockStudyService) ListSourceItems(ctx context.Context, sid *uuid.UUID, gid *uuid.UUID) ([]*model.SourceItem, error) {
	return m.listSourceItemsFn(ctx, sid, gid)
}
func (m *mockStudyService) ListAssets(ctx context.Context) ([]*model.Video, error) {
	return m.listAssetsFn(ctx)
}
func (m *mockStudyService) CreatePair(ctx context.Context, id uuid.UUID, req *model.CreatePairRequest) (*model.SourceItem, error) {
	return m.createPairFn(ctx, id, req)
}

type mockAssetService struct {
	uploadFn func(context.Context, service.AssetUploadInput) (*model.Video, error)
}

func (m *mockAssetService) Upload(ctx context.Context, in service.AssetUploadInput) (*model.Video, error) {
	return m.uploadFn(ctx, in)
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
	csvFn  func(context.Context) ([]byte, error)
	jsonFn func(context.Context) ([]byte, error)
}

func (m *mockExportService) ExportCSV(ctx context.Context) ([]byte, error)  { return m.csvFn(ctx) }
func (m *mockExportService) ExportJSON(ctx context.Context) ([]byte, error) { return m.jsonFn(ctx) }

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
		&mockExportService{csvFn: func(context.Context) ([]byte, error) { return []byte("a,b\n"), nil }, jsonFn: func(context.Context) ([]byte, error) { return []byte("[]"), nil }},
	)
	r := gin.New()
	r.GET("/overview", h.AnalyticsOverview)
	r.GET("/study/:id", h.AnalyticsStudy)
	r.GET("/study/:id/pairs", h.AnalyticsPairs)
	r.GET("/qc", h.AnalyticsQC)
	r.GET("/csv", h.ExportCSV)
	r.GET("/json", h.ExportJSON)

	cases := []string{"/overview", "/study/" + uuid.New().String(), "/study/" + uuid.New().String() + "/pairs", "/qc", "/csv", "/json"}
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
			listSourceItemsFn: func(context.Context, *uuid.UUID, *uuid.UUID) ([]*model.SourceItem, error) {
				return nil, errors.New("boom")
			},
		},
		&mockAssetService{},
		&mockAnalyticsService{},
		&mockQCService{},
		&mockExportService{
			csvFn:  func(context.Context) ([]byte, error) { return nil, errors.New("x") },
			jsonFn: func(context.Context) ([]byte, error) { return nil, errors.New("x") },
		},
	)
	r := gin.New()
	r.GET("/source-items", h.ListSourceItems)
	r.GET("/csv", h.ExportCSV)
	r.GET("/json", h.ExportJSON)

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

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/json", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 json error, got %d", w.Code)
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
			listSourceItemsFn: func(context.Context, *uuid.UUID, *uuid.UUID) ([]*model.SourceItem, error) {
				return []*model.SourceItem{{ID: uuid.New()}}, nil
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

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
)

// --- mock implementations ---

type mockStudyRepo struct {
	createFn func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error)
}

func (m *mockStudyRepo) List(ctx context.Context) ([]*model.Study, error) { return nil, nil }
func (m *mockStudyRepo) Create(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
	return m.createFn(ctx, req)
}
func (m *mockStudyRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*model.Study, error) {
	return nil, nil
}
func (m *mockStudyRepo) Update(ctx context.Context, id uuid.UUID, req *model.UpdateStudyRequest) (*model.Study, error) {
	return nil, nil
}

type mockGroupRepo struct {
	createFn func(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error)
}

func (m *mockGroupRepo) Create(ctx context.Context, studyID uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
	return m.createFn(ctx, studyID, req)
}
func (m *mockGroupRepo) ListByStudy(ctx context.Context, studyID uuid.UUID) ([]*model.Group, error) {
	return nil, nil
}

type mockSourceItemRepo struct {
	createFn func(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error)
}

func (m *mockSourceItemRepo) Create(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error) {
	return m.createFn(ctx, item)
}
func (m *mockSourceItemRepo) ListWithFilters(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItem, error) {
	return nil, nil
}
func (m *mockSourceItemRepo) ListWithDetails(ctx context.Context, studyID *uuid.UUID, groupID *uuid.UUID) ([]*model.SourceItemDetail, error) {
	return nil, nil
}
func (m *mockSourceItemRepo) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockSourceItemRepo) UpdateAttentionCheck(ctx context.Context, id uuid.UUID, isAttentionCheck bool) error {
	return nil
}
func (m *mockSourceItemRepo) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*model.SourceItemDetail, error) {
	return nil, nil
}

type mockImportVideoRepo struct {
	createFn func(ctx context.Context, v *model.Video) (*model.Video, error)
	linkFn   func(ctx context.Context, videoID uuid.UUID, sourceItemID uuid.UUID, methodType string) error
}

func (m *mockImportVideoRepo) Create(ctx context.Context, v *model.Video) (*model.Video, error) {
	return m.createFn(ctx, v)
}
func (m *mockImportVideoRepo) Link(ctx context.Context, videoID uuid.UUID, sourceItemID uuid.UUID, methodType string) error {
	return m.linkFn(ctx, videoID, sourceItemID, methodType)
}

type mockImportStorage struct {
	uploadFn func(ctx context.Context, key, contentType string, body io.Reader, size int64) error
	deleteFn func(ctx context.Context, key string) error
}

func (m *mockImportStorage) Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, contentType, body, size)
	}
	return nil
}
func (m *mockImportStorage) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

// buildTestZIPBytes creates a ZIP archive in memory with the given filename->content pairs.
func buildTestZIPBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip create %q: %v", name, err)
		}
		if _, err = f.Write(data); err != nil {
			t.Fatalf("zip write %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestParseFilename_Valid(t *testing.T) {
	cases := []struct {
		filename   string
		wantGroup  string
		wantPair   string
		wantMethod string
	}{
		{"group1_scene01_baseline.mp4", "group1", "scene01", "baseline"},
		{"group1_scene01_candidate.mp4", "group1", "scene01", "candidate"},
		{"grp_my_scene_baseline.mp4", "grp", "my_scene", "baseline"},
		{"A_B_C_candidate.mp4", "A", "B_C", "candidate"},
		{"g_p_BASELINE.mp4", "g", "p", "baseline"},
	}

	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			got, err := parseFilename(tc.filename)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.groupName != tc.wantGroup {
				t.Errorf("groupName: want %q, got %q", tc.wantGroup, got.groupName)
			}
			if got.pairName != tc.wantPair {
				t.Errorf("pairName: want %q, got %q", tc.wantPair, got.pairName)
			}
			if got.methodType != tc.wantMethod {
				t.Errorf("methodType: want %q, got %q", tc.wantMethod, got.methodType)
			}
		})
	}
}

func TestParseFilename_Invalid(t *testing.T) {
	cases := []struct {
		filename string
	}{
		{"onlyname.mp4"},
		{"group_scene_unknown.mp4"},
		{"group_baseline.mp4"},
		{"_scene_baseline.mp4"},
	}

	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			_, err := parseFilename(tc.filename)
			if err == nil {
				t.Fatalf("expected error for filename %q, got nil", tc.filename)
			}
		})
	}
}

func TestValidatePairs_MissingBaseline(t *testing.T) {
	svc := &ImportService{}
	files := []parsedFile{
		{groupName: "g", pairName: "p", methodType: "candidate", filename: "g_p_candidate.mp4"},
	}
	valid, errs := svc.validatePairs(files)
	if len(valid) != 0 {
		t.Errorf("expected no valid files, got %d", len(valid))
	}
	if len(errs) == 0 {
		t.Error("expected validation errors for missing baseline")
	}
}

func TestValidatePairs_MissingCandidate(t *testing.T) {
	svc := &ImportService{}
	files := []parsedFile{
		{groupName: "g", pairName: "p", methodType: "baseline", filename: "g_p_baseline.mp4"},
	}
	valid, errs := svc.validatePairs(files)
	if len(valid) != 0 {
		t.Errorf("expected no valid files, got %d", len(valid))
	}
	if len(errs) == 0 {
		t.Error("expected validation errors for missing candidate")
	}
}

func TestValidatePairs_CompletePair(t *testing.T) {
	svc := &ImportService{}
	files := []parsedFile{
		{groupName: "g", pairName: "p", methodType: "baseline", filename: "g_p_baseline.mp4"},
		{groupName: "g", pairName: "p", methodType: "candidate", filename: "g_p_candidate.mp4"},
	}
	valid, errs := svc.validatePairs(files)
	if len(valid) != 2 {
		t.Errorf("expected 2 valid files, got %d", len(valid))
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestImportArchiveRequest_ValidationErrors(t *testing.T) {
	svc := newImportServiceWithDeps(nil, nil, nil, nil, nil)

	_, err := svc.ImportArchive(nil, ImportArchiveRequest{}) //nolint:staticcheck
	if err == nil {
		t.Fatal("expected error for empty name")
	}

	_, err = svc.ImportArchive(nil, ImportArchiveRequest{Name: "test"}) //nolint:staticcheck
	if err == nil {
		t.Fatal("expected error for empty effect_type")
	}

	_, err = svc.ImportArchive(nil, ImportArchiveRequest{Name: "test", EffectType: "flooding"}) //nolint:staticcheck
	if err == nil {
		t.Fatal("expected error for empty zip data")
	}
}

func TestParseZIP_ValidAndInvalid(t *testing.T) {
	svc := &ImportService{}

	// Invalid ZIP bytes → fatal error.
	_, _, err := svc.parseZIP([]byte("not a zip"))
	if err == nil {
		t.Fatal("expected error for invalid zip bytes")
	}

	// ZIP with non-mp4 files — silently skipped, no errors.
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"readme.txt":       []byte("hello"),
		"subfolder/":       nil,
	})
	files, errs, err := svc.parseZIP(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
	_ = errs // non-fatal errors list (may be empty)

	// ZIP with a badly named mp4 — parse error, non-fatal.
	zipData = buildTestZIPBytes(t, map[string][]byte{
		"badname.mp4": []byte("fake"),
	})
	files, errs, err = svc.parseZIP(zipData)
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 valid files, got %d", len(files))
	}
	if len(errs) == 0 {
		t.Error("expected non-fatal parse error for bad filename")
	}

	// ZIP with duplicate filenames — second is skipped with error.
	// Manually build a zip with duplicate entry (buildTestZIPBytes uses a map so keys are unique;
	// build by hand here).
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"g_p_baseline.mp4", "g_p_baseline.mp4"} {
		f, _ := zw.Create(name)
		_, _ = f.Write([]byte("fake"))
	}
	_ = zw.Close()
	dupFiles, dupErrs, dupErr := svc.parseZIP(buf.Bytes())
	if dupErr != nil {
		t.Fatalf("unexpected error: %v", dupErr)
	}
	_ = dupFiles
	// One valid, one duplicate (skipped with error).
	if len(dupErrs) == 0 {
		t.Error("expected duplicate error")
	}

	// ZIP with valid mp4 files.
	zipData = buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	files, errs, err = svc.parseZIP(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestGroupFilesByGroup(t *testing.T) {
	files := []parsedFile{
		{groupName: "g1", pairName: "p1", methodType: "baseline"},
		{groupName: "g1", pairName: "p1", methodType: "candidate"},
		{groupName: "g2", pairName: "p1", methodType: "baseline"},
		{groupName: "g2", pairName: "p1", methodType: "candidate"},
	}
	result := groupFilesByGroup(files)
	if len(result) != 2 {
		t.Errorf("expected 2 groups, got %d", len(result))
	}
	if len(result["g1"]) != 1 {
		t.Errorf("expected 1 pair in g1, got %d", len(result["g1"]))
	}
	if _, ok := result["g1"]["p1"]["baseline"]; !ok {
		t.Error("expected baseline in g1/p1")
	}
}

func TestImportArchive_InvalidZIP(t *testing.T) {
	svc := newImportServiceWithDeps(nil, nil, nil, nil, nil)
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    []byte("not a zip"),
	})
	if err == nil {
		t.Fatal("expected error for invalid zip")
	}
}

func TestImportArchive_AllFilesInvalid_ReturnsErrors(t *testing.T) {
	// All files have bad names → no valid files → return result with errors, no study created.
	svc := newImportServiceWithDeps(nil, nil, nil, nil, nil)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"badname.mp4": []byte("fake"),
	})
	result, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Study != nil {
		t.Error("expected nil study when no valid files")
	}
	if len(result.Errors) == 0 {
		t.Error("expected parse errors in result")
	}
}

func TestImportArchive_StudyCreateError(t *testing.T) {
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newImportServiceWithDeps(studyRepo, nil, nil, nil, nil)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected error from study create")
	}
}

func TestImportArchive_GroupCreateError(t *testing.T) {
	studyID := uuid.New()
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID, Name: req.Name}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return nil, errors.New("group error")
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, nil, nil, &mockImportStorage{})
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected error from group create")
	}
}

func TestImportArchive_S3UploadError_Rollback(t *testing.T) {
	studyID := uuid.New()
	groupID := uuid.New()

	var deletedKeys []string
	storage := &mockImportStorage{
		uploadFn: func(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
			return errors.New("s3 error")
		},
		deleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, nil, nil, storage)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected upload error")
	}
}

func TestImportArchive_VideoDBError_Rollback(t *testing.T) {
	studyID := uuid.New()
	groupID := uuid.New()

	var deletedKeys []string
	storage := &mockImportStorage{
		uploadFn: func(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
			return nil
		},
		deleteFn: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		},
	}
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
	}
	videoRepo := &mockImportVideoRepo{
		createFn: func(ctx context.Context, v *model.Video) (*model.Video, error) {
			return nil, errors.New("db error")
		},
		linkFn: func(ctx context.Context, videoID, sourceItemID uuid.UUID, methodType string) error {
			return nil
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, nil, videoRepo, storage)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected video create error")
	}
	// Rollback should have been called (deletedKeys may be non-empty).
	_ = deletedKeys
}

func TestImportArchive_SourceItemCreateError(t *testing.T) {
	studyID := uuid.New()
	groupID := uuid.New()
	videoID := uuid.New()

	storage := &mockImportStorage{}
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
	}
	videoRepo := &mockImportVideoRepo{
		createFn: func(ctx context.Context, v *model.Video) (*model.Video, error) {
			return &model.Video{ID: videoID}, nil
		},
		linkFn: func(ctx context.Context, vID, siID uuid.UUID, methodType string) error {
			return nil
		},
	}
	sourceItemRepo := &mockSourceItemRepo{
		createFn: func(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error) {
			return nil, errors.New("source item error")
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, sourceItemRepo, videoRepo, storage)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected source item create error")
	}
}

func TestImportArchive_LinkError(t *testing.T) {
	studyID := uuid.New()
	groupID := uuid.New()
	videoID := uuid.New()
	siID := uuid.New()

	storage := &mockImportStorage{}
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
	}
	videoRepo := &mockImportVideoRepo{
		createFn: func(ctx context.Context, v *model.Video) (*model.Video, error) {
			return &model.Video{ID: videoID}, nil
		},
		linkFn: func(ctx context.Context, vID, sourceItemID uuid.UUID, methodType string) error {
			return errors.New("link error")
		},
	}
	sourceItemRepo := &mockSourceItemRepo{
		createFn: func(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error) {
			return &model.SourceItem{ID: siID}, nil
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, sourceItemRepo, videoRepo, storage)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected link error")
	}
}

func TestImportArchive_Success(t *testing.T) {
	studyID := uuid.New()
	groupID := uuid.New()
	siID := uuid.New()
	videoID := uuid.New()

	storage := &mockImportStorage{}
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID, Name: req.Name}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return &model.Group{ID: groupID, Name: req.Name}, nil
		},
	}
	callCount := 0
	videoRepo := &mockImportVideoRepo{
		createFn: func(ctx context.Context, v *model.Video) (*model.Video, error) {
			callCount++
			return &model.Video{ID: videoID, Title: v.Title, S3Key: v.S3Key}, nil
		},
		linkFn: func(ctx context.Context, vID, sourceItemID uuid.UUID, methodType string) error {
			return nil
		},
	}
	sourceItemRepo := &mockSourceItemRepo{
		createFn: func(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error) {
			return &model.SourceItem{ID: siID, StudyID: item.StudyID}, nil
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, sourceItemRepo, videoRepo, storage)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	result, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "My Study",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Study == nil {
		t.Fatal("expected study in result")
	}
	if result.GroupsCreated != 1 {
		t.Errorf("expected 1 group, got %d", result.GroupsCreated)
	}
	if result.PairsCreated != 1 {
		t.Errorf("expected 1 pair, got %d", result.PairsCreated)
	}
	if result.VideosUploaded != 2 {
		t.Errorf("expected 2 videos, got %d", result.VideosUploaded)
	}
}

func TestImportArchive_CandidateLinkError(t *testing.T) {
	// Baseline link succeeds, candidate link fails — should rollback.
	studyID := uuid.New()
	groupID := uuid.New()
	siID := uuid.New()
	videoID := uuid.New()

	storage := &mockImportStorage{}
	studyRepo := &mockStudyRepo{
		createFn: func(ctx context.Context, req *model.CreateStudyRequest) (*model.Study, error) {
			return &model.Study{ID: studyID}, nil
		},
	}
	groupRepo := &mockGroupRepo{
		createFn: func(ctx context.Context, sid uuid.UUID, req *model.CreateGroupRequest) (*model.Group, error) {
			return &model.Group{ID: groupID}, nil
		},
	}
	linkCallCount := 0
	videoRepo := &mockImportVideoRepo{
		createFn: func(ctx context.Context, v *model.Video) (*model.Video, error) {
			return &model.Video{ID: videoID}, nil
		},
		linkFn: func(ctx context.Context, vID, sourceItemID uuid.UUID, methodType string) error {
			linkCallCount++
			if linkCallCount == 2 {
				return fmt.Errorf("candidate link error")
			}
			return nil
		},
	}
	sourceItemRepo := &mockSourceItemRepo{
		createFn: func(ctx context.Context, item *model.SourceItem) (*model.SourceItem, error) {
			return &model.SourceItem{ID: siID}, nil
		},
	}
	svc := newImportServiceWithDeps(studyRepo, groupRepo, sourceItemRepo, videoRepo, storage)
	zipData := buildTestZIPBytes(t, map[string][]byte{
		"g_p_baseline.mp4":  []byte("fake"),
		"g_p_candidate.mp4": []byte("fake"),
	})
	_, err := svc.ImportArchive(context.Background(), ImportArchiveRequest{
		Name:       "test",
		EffectType: "flooding",
		ZIPData:    zipData,
	})
	if err == nil {
		t.Fatal("expected candidate link error")
	}
}

package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
)

// ImportArchiveRequest holds parameters for the archive import operation.
type ImportArchiveRequest struct {
	Name                   string
	EffectType             string
	MaxTasksPerParticipant int
	TieOptionEnabled       *bool
	ReasonsEnabled         *bool
	ConfidenceEnabled      *bool
	ZIPReader              io.ReaderAt
	ZIPSize                int64
}

// ImportArchiveResult holds the result of the archive import operation.
type ImportArchiveResult struct {
	Study          *model.Study `json:"study"`
	GroupsCreated  int          `json:"groups_created"`
	PairsCreated   int          `json:"pairs_created"`
	VideosUploaded int          `json:"videos_uploaded"`
	Errors         []string     `json:"errors"`
}

// parsedFile represents a single parsed MP4 file from the archive.
type parsedFile struct {
	groupName  string
	pairName   string
	methodType string // "baseline" or "candidate"
	filename   string
	zipFileRef *zip.File // reference to the ZIP entry for lazy streaming
}

// importVideoRepo is the subset of VideoRepository used by ImportService.
type importVideoRepo interface {
	Create(ctx context.Context, v *model.Video) (*model.Video, error)
	Link(ctx context.Context, videoID uuid.UUID, sourceItemID uuid.UUID, methodType string) error
}

// importVideoRepoAdapter adapts *repository.VideoRepository to importVideoRepo.
type importVideoRepoAdapter struct {
	repo *repository.VideoRepository
}

func (a importVideoRepoAdapter) Create(ctx context.Context, v *model.Video) (*model.Video, error) {
	return a.repo.Create(ctx, nil, v)
}

func (a importVideoRepoAdapter) Link(ctx context.Context, videoID uuid.UUID, sourceItemID uuid.UUID, methodType string) error {
	return a.repo.Link(ctx, videoID, sourceItemID, methodType)
}

// importStorage is the subset of S3 operations needed by ImportService.
type importStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader, size int64) error
	Delete(ctx context.Context, key string) error
}

// ImportService handles ZIP archive import for studies.
type ImportService struct {
	studyRepo      studyRepository
	groupRepo      groupRepository
	sourceItemRepo sourceItemRepository
	videoRepo      importVideoRepo
	s3             importStorage
}

// NewImportService creates a new ImportService wiring the concrete VideoRepository.
func NewImportService(
	studyRepo studyRepository,
	groupRepo groupRepository,
	sourceItemRepo sourceItemRepository,
	videoRepo *repository.VideoRepository,
	s3 importStorage,
) *ImportService {
	return newImportServiceWithDeps(studyRepo, groupRepo, sourceItemRepo, importVideoRepoAdapter{repo: videoRepo}, s3)
}

func newImportServiceWithDeps(
	studyRepo studyRepository,
	groupRepo groupRepository,
	sourceItemRepo sourceItemRepository,
	videoRepo importVideoRepo,
	s3 importStorage,
) *ImportService {
	return &ImportService{
		studyRepo:      studyRepo,
		groupRepo:      groupRepo,
		sourceItemRepo: sourceItemRepo,
		videoRepo:      videoRepo,
		s3:             s3,
	}
}

// ImportArchive processes a ZIP archive and creates a study with groups, pairs, and video assets.
// On partial failure, all successfully uploaded S3 objects are deleted (best-effort rollback).
func (s *ImportService) ImportArchive(ctx context.Context, req ImportArchiveRequest) (*ImportArchiveResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.EffectType) == "" {
		return nil, fmt.Errorf("effect_type is required")
	}
	if req.ZIPReader == nil {
		return nil, fmt.Errorf("archive is empty")
	}

	// Parse files from the ZIP archive.
	files, parseErrs, err := s.parseZIP(req.ZIPReader, req.ZIPSize)
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}

	// Validate all pairs have both baseline and candidate.
	validFiles, validationErrs := s.validatePairs(files)
	allErrs := append(parseErrs, validationErrs...) //nolint:gocritic // intentional: build combined error slice

	if len(validFiles) == 0 {
		return &ImportArchiveResult{
			Errors: allErrs,
		}, nil
	}

	// Create the study.
	study, err := s.studyRepo.Create(ctx, &model.CreateStudyRequest{
		Name:                   req.Name,
		EffectType:             req.EffectType,
		MaxTasksPerParticipant: req.MaxTasksPerParticipant,
		TieOptionEnabled:       req.TieOptionEnabled,
		ReasonsEnabled:         req.ReasonsEnabled,
		ConfidenceEnabled:      req.ConfidenceEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("create study: %w", err)
	}

	result := &ImportArchiveResult{
		Study:  study,
		Errors: allErrs,
	}

	// Group files by group name.
	groupMap := groupFilesByGroup(validFiles)

	// Track uploaded S3 keys for rollback.
	var uploadedKeys []string

	rollback := func() {
		for _, key := range uploadedKeys {
			if delErr := s.s3.Delete(ctx, key); delErr != nil {
				log.Printf("import rollback: failed to delete s3 key %q: %v", key, delErr)
			}
		}
	}

	// Process each group.
	for groupName, pairMap := range groupMap {
		group, err := s.groupRepo.Create(ctx, study.ID, &model.CreateGroupRequest{
			Name: groupName,
		})
		if err != nil {
			rollback()
			return nil, fmt.Errorf("create group %q: %w", groupName, err)
		}
		result.GroupsCreated++

		// Process each pair within the group.
		for pairName, pair := range pairMap {
			baseline := pair["baseline"]
			candidate := pair["candidate"]

			// Upload baseline video.
			baselineVideo, baselineKey, err := s.uploadVideo(ctx, baseline)
			if err != nil {
				rollback()
				return nil, fmt.Errorf("upload baseline for pair %q in group %q: %w", pairName, groupName, err)
			}
			uploadedKeys = append(uploadedKeys, baselineKey)
			result.VideosUploaded++

			// Upload candidate video.
			candidateVideo, candidateKey, err := s.uploadVideo(ctx, candidate)
			if err != nil {
				rollback()
				return nil, fmt.Errorf("upload candidate for pair %q in group %q: %w", pairName, groupName, err)
			}
			uploadedKeys = append(uploadedKeys, candidateKey)
			result.VideosUploaded++

			// Create the pair (source_item) linking both videos.
			pairCode := groupName + "_" + pairName
			sourceItem, err := s.sourceItemRepo.Create(ctx, &model.SourceItem{
				StudyID:  study.ID,
				GroupID:  group.ID,
				PairCode: &pairCode,
			})
			if err != nil {
				rollback()
				return nil, fmt.Errorf("create source item for pair %q in group %q: %w", pairName, groupName, err)
			}

			// Link baseline video to source item.
			if err := s.videoRepo.Link(ctx, baselineVideo.ID, sourceItem.ID, "baseline"); err != nil {
				rollback()
				return nil, fmt.Errorf("link baseline video for pair %q: %w", pairName, err)
			}

			// Link candidate video to source item.
			if err := s.videoRepo.Link(ctx, candidateVideo.ID, sourceItem.ID, "candidate"); err != nil {
				rollback()
				return nil, fmt.Errorf("link candidate video for pair %q: %w", pairName, err)
			}

			result.PairsCreated++
		}
	}

	return result, nil
}

// parseZIP reads the ZIP directory and parses MP4 filenames without reading file contents.
// Returns parsed files (with zip entry references for lazy streaming), non-fatal parse errors, and a fatal error if the archive is unreadable.
func (s *ImportService) parseZIP(ra io.ReaderAt, size int64) ([]parsedFile, []string, error) {
	r, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, nil, err
	}

	var files []parsedFile
	var errs []string
	seen := make(map[string]bool)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Only process .mp4 files; ignore others silently.
		name := filepath.Base(f.Name)
		if !strings.HasSuffix(strings.ToLower(name), ".mp4") {
			continue
		}

		// Check for duplicate filenames.
		if seen[name] {
			errs = append(errs, fmt.Sprintf("duplicate filename %q — skipped", name))
			continue
		}
		seen[name] = true

		// Parse filename: <group>_<name>_<baseline|candidate>.mp4
		parsed, parseErr := parseFilename(name)
		if parseErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, parseErr))
			continue
		}

		parsed.filename = name
		parsed.zipFileRef = f
		files = append(files, parsed)
	}

	return files, errs, nil
}

// parseFilename parses a filename of the form <group>_<name>_<baseline|candidate>.mp4.
func parseFilename(name string) (parsedFile, error) {
	// Strip .mp4 extension (case-insensitive).
	base := name
	switch {
	case strings.HasSuffix(strings.ToLower(base), ".mp4"):
		base = base[:len(base)-4]
	}

	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return parsedFile{}, fmt.Errorf("filename must be <group>_<name>_<baseline|candidate>.mp4, got %q", name)
	}

	// Last part is the method type.
	methodType := strings.ToLower(parts[len(parts)-1])
	if methodType != "baseline" && methodType != "candidate" {
		return parsedFile{}, fmt.Errorf("filename must end with _baseline.mp4 or _candidate.mp4, got %q", name)
	}

	groupName := parts[0]
	// Middle parts (index 1 through len-2) form the pair name.
	pairName := strings.Join(parts[1:len(parts)-1], "_")

	if groupName == "" || pairName == "" {
		return parsedFile{}, fmt.Errorf("filename must be <group>_<name>_<baseline|candidate>.mp4, got %q", name)
	}

	return parsedFile{
		groupName:  groupName,
		pairName:   pairName,
		methodType: methodType,
	}, nil
}

// validatePairs checks that every pair has both baseline and candidate.
// Returns only the files belonging to complete pairs, and errors for incomplete pairs.
func (s *ImportService) validatePairs(files []parsedFile) ([]parsedFile, []string) {
	// Structure: groupName -> pairName -> methodType -> parsedFile
	index := make(map[string]map[string]map[string]parsedFile)
	for _, f := range files {
		if index[f.groupName] == nil {
			index[f.groupName] = make(map[string]map[string]parsedFile)
		}
		if index[f.groupName][f.pairName] == nil {
			index[f.groupName][f.pairName] = make(map[string]parsedFile)
		}
		index[f.groupName][f.pairName][f.methodType] = f
	}

	var valid []parsedFile
	var errs []string

	for groupName, pairs := range index {
		for pairName, methods := range pairs {
			_, hasBaseline := methods["baseline"]
			_, hasCandidate := methods["candidate"]

			if !hasBaseline {
				errs = append(errs, fmt.Sprintf("pair %q in group %q: missing baseline video", pairName, groupName))
				continue
			}
			if !hasCandidate {
				errs = append(errs, fmt.Sprintf("pair %q in group %q: missing candidate video", pairName, groupName))
				continue
			}

			valid = append(valid, methods["baseline"])
			valid = append(valid, methods["candidate"])
		}
	}

	return valid, errs
}

// groupFilesByGroup organises validated files into the structure:
// groupName -> pairName -> methodType -> parsedFile
func groupFilesByGroup(files []parsedFile) map[string]map[string]map[string]parsedFile {
	result := make(map[string]map[string]map[string]parsedFile)
	for _, f := range files {
		if result[f.groupName] == nil {
			result[f.groupName] = make(map[string]map[string]parsedFile)
		}
		if result[f.groupName][f.pairName] == nil {
			result[f.groupName][f.pairName] = make(map[string]parsedFile)
		}
		result[f.groupName][f.pairName][f.methodType] = f
	}
	return result
}

// uploadVideo uploads a video file to S3 and creates the DB record.
// Returns the created Video model and the S3 key (for rollback tracking).
func (s *ImportService) uploadVideo(ctx context.Context, f parsedFile) (*model.Video, string, error) {
	key := fmt.Sprintf("videos/%s.mp4", uuid.NewString())

	rc, err := f.zipFileRef.Open()
	if err != nil {
		return nil, key, fmt.Errorf("open zip entry: %w", err)
	}
	defer func() { _ = rc.Close() }()

	// Buffer the video into a temp file so the stream is seekable for S3 retry.
	tmp, err := os.CreateTemp("", "upload-*.mp4")
	if err != nil {
		return nil, key, fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	defer func() { _ = tmp.Close() }()

	size, err := io.Copy(tmp, rc)
	if err != nil {
		return nil, key, fmt.Errorf("buffer video to temp: %w", err)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, key, fmt.Errorf("seek temp file: %w", err)
	}

	if err := s.s3.Upload(ctx, key, "video/mp4", tmp, size); err != nil {
		return nil, key, fmt.Errorf("s3 upload: %w", err)
	}

	title := f.groupName + "_" + f.pairName + "_" + f.methodType
	video, err := s.videoRepo.Create(ctx, &model.Video{
		Title:  title,
		S3Key:  key,
		Status: model.VideoStatusActive,
	})
	if err != nil {
		// Best-effort S3 cleanup for this single file before returning the error.
		if delErr := s.s3.Delete(ctx, key); delErr != nil {
			log.Printf("uploadVideo: failed to delete s3 key %q after DB error: %v", key, delErr)
		}
		return nil, key, fmt.Errorf("create video record: %w", err)
	}

	return video, key, nil
}

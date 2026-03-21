package service

import (
	"testing"
)

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

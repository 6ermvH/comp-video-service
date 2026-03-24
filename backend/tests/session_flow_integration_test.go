//go:build integration

package tests

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/service"
)

func TestSessionStartAndComplete_NoTasks(t *testing.T) {
	db := mustOpenDB(t)
	ctx := context.Background()

	studyID := mustCreateStudy(t, ctx, db, "int-start-complete", "active")

	studyRepo := repository.NewStudyRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	sourceItemRepo := repository.NewSourceItemRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	pairRepo := repository.NewPairPresentationRepository(db)
	responseRepo := repository.NewResponseRepository(db)

	assignmentSvc := service.NewAssignmentService(sourceItemRepo, groupRepo, videoRepo, pairRepo)
	qcSvc := service.NewQCService(responseRepo, participantRepo)
	sessionSvc := service.NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, nil)

	started, err := sessionSvc.Start(ctx, &model.StartSessionRequest{StudyID: studyID, DeviceType: "desktop"})
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	if started.Assigned != 0 {
		t.Fatalf("expected assigned=0, got %d", started.Assigned)
	}
	if started.FirstTask != nil {
		t.Fatalf("expected first_task=nil when no source items")
	}
	if started.Meta.StudyID != studyID {
		t.Fatalf("expected meta.study_id=%s, got %s", studyID, started.Meta.StudyID)
	}

	completed, err := sessionSvc.Complete(ctx, started.SessionToken)
	if err != nil {
		t.Fatalf("session complete: %v", err)
	}
	if completed.CompletionCode == "" {
		t.Fatal("expected non-empty completion code")
	}

	var completedAt *time.Time
	err = db.QueryRow(ctx, `SELECT completed_at FROM participants WHERE session_token=$1`, started.SessionToken).Scan(&completedAt)
	if err != nil {
		t.Fatalf("select completed_at: %v", err)
	}
	if completedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestSaveResponseAndExportCSVHeader(t *testing.T) {
	db := mustOpenDB(t)
	ctx := context.Background()

	studyID := mustCreateStudy(t, ctx, db, "int-response-export", "active")
	groupID := mustCreateGroup(t, ctx, db, studyID, "int-group")
	sourceItemID := mustCreateSourceItem(t, ctx, db, studyID, groupID)
	leftID := mustCreateVideoAsset(t, ctx, db, sourceItemID, "baseline", "videos/int-left.mp4")
	rightID := mustCreateVideoAsset(t, ctx, db, sourceItemID, "candidate", "videos/int-right.mp4")
	participantID, token := mustCreateParticipant(t, ctx, db, studyID)
	pairID := mustCreatePairPresentation(t, ctx, db, participantID, sourceItemID, leftID, rightID)

	studyRepo := repository.NewStudyRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	pairRepo := repository.NewPairPresentationRepository(db)
	responseRepo := repository.NewResponseRepository(db)
	sourceItemRepo := repository.NewSourceItemRepository(db)

	assignmentSvc := service.NewAssignmentService(sourceItemRepo, groupRepo, videoRepo, pairRepo)
	qcSvc := service.NewQCService(responseRepo, participantRepo)
	sessionSvc := service.NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, nil)

	conf := 4
	ms := 2200
	resp, err := sessionSvc.SaveResponse(ctx, pairID, &model.TaskResponseRequest{
		Choice:         "left",
		ReasonCodes:    []string{"realism", "physics"},
		Confidence:     &conf,
		ResponseTimeMS: &ms,
		ReplayCount:    1,
	})
	if err != nil {
		t.Fatalf("save response: %v", err)
	}
	if resp.ID == uuid.Nil {
		t.Fatal("expected response id")
	}

	exportSvc := service.NewExportService(db)
	csvPayload, err := exportSvc.ExportCSV(ctx)
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}

	records, err := csv.NewReader(bytes.NewReader(csvPayload)).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) < 2 {
		t.Fatalf("expected at least header+1 row, got %d", len(records))
	}

	header := records[0]
	if len(header) != 20 {
		t.Fatalf("expected 20 columns, got %d", len(header))
	}
	first := strings.TrimPrefix(header[0], "\xEF\xBB\xBF") // strip UTF-8 BOM
	if first != "response_id" || header[19] != "created_at" {
		t.Fatalf("unexpected export header: first=%q last=%q", header[0], header[19])
	}

	_ = token
}

func TestAssignmentPrefersUnderTargetSourceItems(t *testing.T) {
	db := mustOpenDB(t)
	ctx := context.Background()

	studyID := mustCreateStudy(t, ctx, db, "int-assignment-balance", "active")
	groupID := mustCreateGroup(t, ctx, db, studyID, "priority-group")
	sourceUnderTarget := mustCreateSourceItem(t, ctx, db, studyID, groupID)
	sourceSaturated := mustCreateSourceItem(t, ctx, db, studyID, groupID)

	underLeft := mustCreateVideoAsset(t, ctx, db, sourceUnderTarget, "baseline", "videos/int-under-left.mp4")
	underRight := mustCreateVideoAsset(t, ctx, db, sourceUnderTarget, "candidate", "videos/int-under-right.mp4")
	satLeft := mustCreateVideoAsset(t, ctx, db, sourceSaturated, "baseline", "videos/int-sat-left.mp4")
	satRight := mustCreateVideoAsset(t, ctx, db, sourceSaturated, "candidate", "videos/int-sat-right.mp4")
	_ = underLeft
	_ = underRight

	// Saturate one source item above target_votes_per_pair(default 10).
	for i := 0; i < 12; i++ {
		pID, _ := mustCreateParticipant(t, ctx, db, studyID)
		ppID := mustCreatePairPresentation(t, ctx, db, pID, sourceSaturated, satLeft, satRight)
		if _, err := db.Exec(ctx, `
			INSERT INTO responses (participant_id, pair_presentation_id, choice, replay_count)
			VALUES ($1, $2, 'left', 0)`, pID, ppID); err != nil {
			t.Fatalf("seed saturated responses: %v", err)
		}
	}

	targetParticipant, _ := mustCreateParticipant(t, ctx, db, studyID)

	sourceItemRepo := repository.NewSourceItemRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	videoRepo := repository.NewVideoRepository(db)
	pairRepo := repository.NewPairPresentationRepository(db)
	assignmentSvc := service.NewAssignmentService(sourceItemRepo, groupRepo, videoRepo, pairRepo)

	created, err := assignmentSvc.AssignForParticipant(ctx, targetParticipant, studyID, 1)
	if err != nil {
		t.Fatalf("assign for participant: %v", err)
	}
	if created != 1 {
		t.Fatalf("expected one assigned task, got %d", created)
	}

	var assignedSource uuid.UUID
	err = db.QueryRow(ctx, `
		SELECT source_item_id
		FROM pair_presentations
		WHERE participant_id = $1
		ORDER BY task_order ASC
		LIMIT 1`, targetParticipant).Scan(&assignedSource)
	if err != nil {
		t.Fatalf("query assigned source item: %v", err)
	}

	if assignedSource != sourceUnderTarget {
		t.Fatalf("expected under-target source item %s, got %s", sourceUnderTarget, assignedSource)
	}
}

func mustCreateStudy(t *testing.T, ctx context.Context, db *pgxpool.Pool, suffix, status string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO studies (name, effect_type, status)
		VALUES ($1, 'flooding', $2)
		RETURNING id`, "Integration "+suffix+" "+uuid.NewString(), status).Scan(&id)
	if err != nil {
		t.Fatalf("create study: %v", err)
	}
	return id
}

func mustCreateGroup(t *testing.T, ctx context.Context, db *pgxpool.Pool, studyID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO groups (study_id, name)
		VALUES ($1, $2)
		RETURNING id`, studyID, name+"-"+uuid.NewString()).Scan(&id)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	return id
}

func mustCreateSourceItem(t *testing.T, ctx context.Context, db *pgxpool.Pool, studyID, groupID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO source_items (study_id, group_id, pair_code)
		VALUES ($1, $2, $3)
		RETURNING id`, studyID, groupID, "pair-"+uuid.NewString()).Scan(&id)
	if err != nil {
		t.Fatalf("create source_item: %v", err)
	}
	return id
}

func mustCreateVideoAsset(t *testing.T, ctx context.Context, db *pgxpool.Pool, sourceItemID uuid.UUID, methodType, s3Key string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO video_assets (source_item_id, method_type, title, description, s3_key, status)
		VALUES ($1, $2, $3, 'integration', $4, 'active')
		RETURNING id`, sourceItemID, methodType, methodType+"-"+uuid.NewString(), s3Key+"-"+uuid.NewString()).Scan(&id)
	if err != nil {
		t.Fatalf("create video_asset: %v", err)
	}
	return id
}

func mustCreateParticipant(t *testing.T, ctx context.Context, db *pgxpool.Pool, studyID uuid.UUID) (uuid.UUID, string) {
	t.Helper()
	var id uuid.UUID
	token := "session-" + uuid.NewString()
	err := db.QueryRow(ctx, `
		INSERT INTO participants (session_token, study_id)
		VALUES ($1, $2)
		RETURNING id`, token, studyID).Scan(&id)
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}
	return id, token
}

func mustCreatePairPresentation(t *testing.T, ctx context.Context, db *pgxpool.Pool, participantID, sourceItemID, leftID, rightID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO pair_presentations (
			participant_id, source_item_id, left_asset_id, right_asset_id,
			left_method_type, right_method_type, task_order
		)
		VALUES ($1,$2,$3,$4,'baseline','candidate',1)
		RETURNING id`, participantID, sourceItemID, leftID, rightID).Scan(&id)
	if err != nil {
		t.Fatalf("create pair_presentation: %v", err)
	}
	return id
}

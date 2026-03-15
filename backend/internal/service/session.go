package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
	"comp-video-service/backend/internal/storage"
)

// SessionStartResult is API response for /session/start.
type SessionStartResult struct {
	SessionToken string             `json:"session_token"`
	Assigned     int                `json:"assigned"`
	Meta         SessionStartMeta   `json:"meta"`
	FirstTask    *model.TaskPayload `json:"first_task,omitempty"`
}

// SessionCompleteResult is API response for /session/:token/complete.
type SessionCompleteResult struct {
	CompletionCode string `json:"completion_code"`
}

// SessionStartMeta contains study-level toggles needed by frontend flow.
type SessionStartMeta struct {
	StudyID                uuid.UUID `json:"study_id"`
	StudyName              string    `json:"study_name"`
	EffectType             string    `json:"effect_type"`
	InstructionsText       *string   `json:"instructions_text,omitempty"`
	MaxTasksPerParticipant int       `json:"max_tasks_per_participant"`
	TieOptionEnabled       bool      `json:"tie_option_enabled"`
	ReasonsEnabled         bool      `json:"reasons_enabled"`
	ConfidenceEnabled      bool      `json:"confidence_enabled"`
}

// SessionService controls respondent lifecycle.
type SessionService struct {
	studyRepo       *repository.StudyRepository
	participantRepo *repository.ParticipantRepository
	pairRepo        *repository.PairPresentationRepository
	videoRepo       *repository.VideoRepository
	responseRepo    *repository.ResponseRepository
	assignmentSvc   *AssignmentService
	qcSvc           *QCService
	s3              *storage.S3Client
	presignTTL      time.Duration
}

func NewSessionService(
	studyRepo *repository.StudyRepository,
	participantRepo *repository.ParticipantRepository,
	pairRepo *repository.PairPresentationRepository,
	videoRepo *repository.VideoRepository,
	responseRepo *repository.ResponseRepository,
	assignmentSvc *AssignmentService,
	qcSvc *QCService,
	s3 *storage.S3Client,
) *SessionService {
	return &SessionService{
		studyRepo:       studyRepo,
		participantRepo: participantRepo,
		pairRepo:        pairRepo,
		videoRepo:       videoRepo,
		responseRepo:    responseRepo,
		assignmentSvc:   assignmentSvc,
		qcSvc:           qcSvc,
		s3:              s3,
		presignTTL:      time.Hour,
	}
}

func (s *SessionService) Start(ctx context.Context, req *model.StartSessionRequest) (*SessionStartResult, error) {
	study, err := s.studyRepo.GetByID(ctx, req.StudyID)
	if err != nil {
		return nil, fmt.Errorf("study not found: %w", err)
	}
	if study.Status != "active" {
		return nil, fmt.Errorf("study is not active")
	}

	token := uuid.NewString()
	participant, err := s.participantRepo.Create(ctx, &model.Participant{
		SessionToken: token,
		StudyID:      req.StudyID,
		DeviceType:   nilIfEmpty(req.DeviceType),
		Browser:      nilIfEmpty(req.Browser),
		Role:         nilIfEmpty(req.Role),
		Experience:   nilIfEmpty(req.Experience),
	})
	if err != nil {
		return nil, err
	}

	assigned, err := s.assignmentSvc.AssignForParticipant(ctx, participant.ID, req.StudyID, study.MaxTasksPerParticipant)
	if err != nil {
		return nil, err
	}

	var firstTask *model.TaskPayload
	if assigned > 0 {
		firstTask, err = s.NextTask(ctx, token)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	meta := SessionStartMeta{
		StudyID:                study.ID,
		StudyName:              study.Name,
		EffectType:             study.EffectType,
		InstructionsText:       study.InstructionsText,
		MaxTasksPerParticipant: study.MaxTasksPerParticipant,
		TieOptionEnabled:       study.TieOptionEnabled,
		ReasonsEnabled:         study.ReasonsEnabled,
		ConfidenceEnabled:      study.ConfidenceEnabled,
	}

	return &SessionStartResult{
		SessionToken: token,
		Assigned:     assigned,
		Meta:         meta,
		FirstTask:    firstTask,
	}, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *SessionService) NextTask(ctx context.Context, token string) (*model.TaskPayload, error) {
	pp, err := s.pairRepo.GetNextPendingByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	left, err := s.videoRepo.GetByID(ctx, pp.LeftAssetID)
	if err != nil {
		return nil, err
	}
	right, err := s.videoRepo.GetByID(ctx, pp.RightAssetID)
	if err != nil {
		return nil, err
	}

	left.PresignedURL, err = s.s3.PresignedURL(ctx, left.S3Key, s.presignTTL)
	if err != nil {
		return nil, err
	}
	right.PresignedURL, err = s.s3.PresignedURL(ctx, right.S3Key, s.presignTTL)
	if err != nil {
		return nil, err
	}

	return &model.TaskPayload{
		PresentationID:   pp.ID,
		SourceItemID:     pp.SourceItemID,
		TaskOrder:        pp.TaskOrder,
		IsAttentionCheck: pp.IsAttentionCheck,
		IsPractice:       pp.IsPractice,
		Left:             left,
		Right:            right,
	}, nil
}

func (s *SessionService) Complete(ctx context.Context, token string) (*SessionCompleteResult, error) {
	participant, err := s.participantRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := s.qcSvc.EvaluateParticipantFinalFlag(ctx, participant.ID); err != nil {
		return nil, err
	}

	if err := s.participantRepo.CompleteByToken(ctx, token); err != nil {
		return nil, err
	}
	code := fmt.Sprintf("CVS-%s", token[:8])
	return &SessionCompleteResult{CompletionCode: code}, nil
}

func (s *SessionService) SaveResponse(
	ctx context.Context,
	presentationID uuid.UUID,
	req *model.TaskResponseRequest,
) (*model.Response, error) {
	pp, err := s.pairRepo.GetByID(ctx, presentationID)
	if err != nil {
		return nil, err
	}

	choice := req.Choice
	if choice != "left" && choice != "right" && choice != "tie" {
		return nil, fmt.Errorf("invalid choice")
	}

	resp, err := s.responseRepo.Create(ctx, &model.Response{
		ParticipantID:      pp.ParticipantID,
		PairPresentationID: pp.ID,
		Choice:             choice,
		ReasonCodes:        req.ReasonCodes,
		Confidence:         req.Confidence,
		ResponseTimeMS:     req.ResponseTimeMS,
		ReplayCount:        req.ReplayCount,
	})
	if err != nil {
		return nil, err
	}

	if err := s.qcSvc.UpdateParticipantFlag(ctx, pp.ParticipantID, req.ResponseTimeMS); err != nil {
		return nil, err
	}
	return resp, nil
}

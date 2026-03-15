package service

import (
	"context"

	"github.com/google/uuid"
)

// QCReport is a lightweight quality control snapshot.
type QCReport struct {
	TotalResponses          int64 `json:"total_responses"`
	FastResponses           int64 `json:"fast_responses"`
	StraightLiningProfiles  int64 `json:"straight_lining_profiles"`
	FastResponseThresholdMS int   `json:"fast_response_threshold_ms"`
}

// QCService provides respondent quality checks.
type QCService struct {
	responseRepo    qcResponseRepository
	participantRepo qcParticipantRepository
}

type qcResponseRepository interface {
	CountTotal(ctx context.Context) (int64, error)
	CountFastResponses(ctx context.Context, thresholdMS int) (int64, error)
	StraightLiningParticipants(ctx context.Context) (int64, error)
	CountByParticipant(ctx context.Context, participantID uuid.UUID) (int64, error)
	CountFastByParticipant(ctx context.Context, participantID uuid.UUID, thresholdMS int) (int64, error)
	AttentionCheckStats(ctx context.Context, participantID uuid.UUID) (int64, int64, error)
}

type qcParticipantRepository interface {
	UpdateQualityFlag(ctx context.Context, participantID uuid.UUID, qualityFlag string) error
}

//go:generate go run go.uber.org/mock/mockgen -source=qc.go -destination=qc_mocks_test.go -package=service

func NewQCService(responseRepo qcResponseRepository, participantRepo qcParticipantRepository) *QCService {
	return &QCService{responseRepo: responseRepo, participantRepo: participantRepo}
}

func (s *QCService) BuildReport(ctx context.Context) (*QCReport, error) {
	total, err := s.responseRepo.CountTotal(ctx)
	if err != nil {
		return nil, err
	}
	fastThreshold := 1500
	fast, err := s.responseRepo.CountFastResponses(ctx, fastThreshold)
	if err != nil {
		return nil, err
	}
	straight, err := s.responseRepo.StraightLiningParticipants(ctx)
	if err != nil {
		return nil, err
	}
	return &QCReport{
		TotalResponses:          total,
		FastResponses:           fast,
		StraightLiningProfiles:  straight,
		FastResponseThresholdMS: fastThreshold,
	}, nil
}

// UpdateParticipantFlag labels participant quality based on a simple response-time heuristic.
func (s *QCService) UpdateParticipantFlag(ctx context.Context, participantID uuid.UUID, responseTimeMS *int) error {
	if responseTimeMS == nil {
		return nil
	}
	if *responseTimeMS < 500 {
		return s.participantRepo.UpdateQualityFlag(ctx, participantID, "suspect")
	}
	return nil
}

// EvaluateParticipantFinalFlag runs QC checks when session is completed.
func (s *QCService) EvaluateParticipantFinalFlag(ctx context.Context, participantID uuid.UUID) error {
	total, err := s.responseRepo.CountByParticipant(ctx, participantID)
	if err != nil {
		return err
	}
	if total == 0 {
		return nil
	}

	fastThreshold := 1500
	fastCount, err := s.responseRepo.CountFastByParticipant(ctx, participantID, fastThreshold)
	if err != nil {
		return err
	}
	attentionTotal, attentionFailed, err := s.responseRepo.AttentionCheckStats(ctx, participantID)
	if err != nil {
		return err
	}

	// Conservative heuristic:
	// - flagged: any failed attention-check or very high fast-response ratio
	// - suspect: moderately high fast-response ratio
	fastRatio := float64(fastCount) / float64(total)
	if attentionTotal > 0 && attentionFailed > 0 {
		return s.participantRepo.UpdateQualityFlag(ctx, participantID, "flagged")
	}
	if fastRatio >= 0.6 {
		return s.participantRepo.UpdateQualityFlag(ctx, participantID, "flagged")
	}
	if fastRatio >= 0.35 {
		return s.participantRepo.UpdateQualityFlag(ctx, participantID, "suspect")
	}
	return s.participantRepo.UpdateQualityFlag(ctx, participantID, "ok")
}

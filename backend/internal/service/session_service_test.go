package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/mock/gomock"

	"comp-video-service/backend/internal/model"
)

func TestSessionStartValidationAndSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMocksessionStudyRepository(ctrl)
	participantRepo := NewMocksessionParticipantRepository(ctrl)
	pairRepo := NewMocksessionPairRepository(ctrl)
	videoRepo := NewMocksessionVideoRepository(ctrl)
	responseRepo := NewMocksessionResponseRepository(ctrl)
	assignmentSvc := NewMocksessionAssignmentService(ctrl)
	qcSvc := NewMocksessionQCService(ctrl)
	s3 := NewMocksessionStorage(ctrl)
	svc := NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, s3)

	studyID := uuid.New()
	studyRepo.EXPECT().GetByID(gomock.Any(), studyID).Return(&model.Study{ID: studyID, Status: "paused"}, nil)
	if _, err := svc.Start(context.Background(), &model.StartSessionRequest{StudyID: studyID}); err == nil {
		t.Fatal("expected inactive study error")
	}

	studyRepo.EXPECT().GetByID(gomock.Any(), studyID).Return(&model.Study{ID: studyID, Status: "active", Name: "S"}, nil)
	participantRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.Participant{ID: uuid.New(), SessionToken: "tok"}, nil)
	assignmentSvc.EXPECT().AssignForParticipant(gomock.Any(), gomock.Any(), studyID, 0).Return(0, nil)
	result, err := svc.Start(context.Background(), &model.StartSessionRequest{StudyID: studyID})
	if err != nil {
		t.Fatalf("Start err: %v", err)
	}
	if result.Assigned != 0 {
		t.Fatalf("expected assigned=0 got %d", result.Assigned)
	}
}

func TestSessionNextTaskAndSaveResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMocksessionStudyRepository(ctrl)
	participantRepo := NewMocksessionParticipantRepository(ctrl)
	pairRepo := NewMocksessionPairRepository(ctrl)
	videoRepo := NewMocksessionVideoRepository(ctrl)
	responseRepo := NewMocksessionResponseRepository(ctrl)
	assignmentSvc := NewMocksessionAssignmentService(ctrl)
	qcSvc := NewMocksessionQCService(ctrl)
	s3 := NewMocksessionStorage(ctrl)
	svc := NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, s3)

	pid := uuid.New()
	leftID := uuid.New()
	rightID := uuid.New()
	pairRepo.EXPECT().GetNextPendingByToken(gomock.Any(), "tok").Return(&model.PairPresentation{ID: pid, LeftAssetID: leftID, RightAssetID: rightID}, nil)
	videoRepo.EXPECT().GetByID(gomock.Any(), leftID).Return(&model.Video{ID: leftID, S3Key: "l"}, nil)
	videoRepo.EXPECT().GetByID(gomock.Any(), rightID).Return(&model.Video{ID: rightID, S3Key: "r"}, nil)
	s3.EXPECT().PresignedURL(gomock.Any(), "l", gomock.Any()).Return("u1", nil)
	s3.EXPECT().PresignedURL(gomock.Any(), "r", gomock.Any()).Return("u2", nil)

	if _, err := svc.NextTask(context.Background(), "tok"); err != nil {
		t.Fatalf("NextTask err: %v", err)
	}

	pairRepo.EXPECT().GetByID(gomock.Any(), pid).Return(&model.PairPresentation{ID: pid, ParticipantID: uuid.New()}, nil)
	if _, err := svc.SaveResponse(context.Background(), pid, &model.TaskResponseRequest{Choice: "bad"}); err == nil {
		t.Fatal("expected invalid choice error")
	}
}

func TestSessionComplete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMocksessionStudyRepository(ctrl)
	participantRepo := NewMocksessionParticipantRepository(ctrl)
	pairRepo := NewMocksessionPairRepository(ctrl)
	videoRepo := NewMocksessionVideoRepository(ctrl)
	responseRepo := NewMocksessionResponseRepository(ctrl)
	assignmentSvc := NewMocksessionAssignmentService(ctrl)
	qcSvc := NewMocksessionQCService(ctrl)
	s3 := NewMocksessionStorage(ctrl)
	svc := NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, s3)

	pid := uuid.New()
	token := "token-12345678"
	participantRepo.EXPECT().GetByToken(gomock.Any(), token).Return(&model.Participant{ID: pid}, nil)
	qcSvc.EXPECT().EvaluateParticipantFinalFlag(gomock.Any(), pid).Return(nil)
	participantRepo.EXPECT().CompleteByToken(gomock.Any(), token).Return(nil)
	if _, err := svc.Complete(context.Background(), token); err != nil {
		t.Fatalf("Complete err: %v", err)
	}

	participantRepo.EXPECT().GetByToken(gomock.Any(), "tok-xxxxxxxx").Return(nil, errors.New("x"))
	if _, err := svc.Complete(context.Background(), "tok-xxxxxxxx"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSessionStartHandlesNoRowsOnPrefetch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	studyRepo := NewMocksessionStudyRepository(ctrl)
	participantRepo := NewMocksessionParticipantRepository(ctrl)
	pairRepo := NewMocksessionPairRepository(ctrl)
	videoRepo := NewMocksessionVideoRepository(ctrl)
	responseRepo := NewMocksessionResponseRepository(ctrl)
	assignmentSvc := NewMocksessionAssignmentService(ctrl)
	qcSvc := NewMocksessionQCService(ctrl)
	s3 := NewMocksessionStorage(ctrl)
	svc := NewSessionService(studyRepo, participantRepo, pairRepo, videoRepo, responseRepo, assignmentSvc, qcSvc, s3)

	studyID := uuid.New()
	studyRepo.EXPECT().GetByID(gomock.Any(), studyID).Return(&model.Study{ID: studyID, Status: "active"}, nil)
	participantRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&model.Participant{ID: uuid.New(), SessionToken: "tok"}, nil)
	assignmentSvc.EXPECT().AssignForParticipant(gomock.Any(), gomock.Any(), studyID, 0).Return(1, nil)
	pairRepo.EXPECT().GetNextPendingByToken(gomock.Any(), gomock.Any()).Return(nil, pgx.ErrNoRows)

	if _, err := svc.Start(context.Background(), &model.StartSessionRequest{StudyID: studyID}); err != nil {
		t.Fatalf("Start err: %v", err)
	}
}

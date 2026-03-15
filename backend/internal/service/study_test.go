package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestStudyServiceUpdateStatus_Invalid(t *testing.T) {
	svc := &StudyService{}
	_, err := svc.UpdateStatus(context.Background(), uuid.New(), "INVALID")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

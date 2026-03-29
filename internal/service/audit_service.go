package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"load_balancing_project_auth/internal/model"
	"load_balancing_project_auth/internal/repository"
)

type AuditService struct {
	repository repository.AuditLogRepository
}

func NewAuditService(repository repository.AuditLogRepository) *AuditService {
	return &AuditService{repository: repository}
}

func (s *AuditService) LogAuthEvent(ctx context.Context, userID *string, eventType string, metadata model.RequestMetadata, extra map[string]any) {
	payload, err := json.Marshal(extra)
	if err != nil {
		log.Printf("audit log marshal failed: %v", err)
		return
	}

	auditLog := model.AuditLog{
		ID:        uuid.NewString(),
		UserID:    userID,
		EventType: eventType,
		IPAddress: metadata.IPAddress,
		UserAgent: metadata.UserAgent,
		Metadata:  payload,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repository.Create(ctx, auditLog); err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}

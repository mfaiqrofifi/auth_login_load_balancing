package service

import (
	"time"

	"load_balancing_project_auth/internal/repository"
)

type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthService struct {
	systemRepository *repository.SystemRepository
}

func NewHealthService(systemRepository *repository.SystemRepository) *HealthService {
	return &HealthService{
		systemRepository: systemRepository,
	}
}

func (s *HealthService) Check() HealthStatus {
	return HealthStatus{
		Status:    "ok",
		Timestamp: s.systemRepository.CurrentTime(),
	}
}

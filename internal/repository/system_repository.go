package repository

import "time"

type SystemRepository struct{}

func NewSystemRepository() *SystemRepository {
	return &SystemRepository{}
}

func (r *SystemRepository) CurrentTime() time.Time {
	return time.Now().UTC()
}

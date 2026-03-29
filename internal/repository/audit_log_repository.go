package repository

import (
	"context"
	"database/sql"

	"load_balancing_project_auth/internal/model"
)

type AuditLogRepository interface {
	Create(ctx context.Context, auditLog model.AuditLog) error
}

type PostgresAuditLogRepository struct {
	db *sql.DB
}

func NewPostgresAuditLogRepository(db *sql.DB) *PostgresAuditLogRepository {
	return &PostgresAuditLogRepository{db: db}
}

func (r *PostgresAuditLogRepository) Create(ctx context.Context, auditLog model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, user_id, event_type, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		auditLog.ID,
		auditLog.UserID,
		auditLog.EventType,
		auditLog.IPAddress,
		auditLog.UserAgent,
		auditLog.Metadata,
		auditLog.CreatedAt,
	)
	return err
}

package repository

import (
	"context"
	"database/sql"

	"load_balancing_project_auth/internal/model"
)

type SessionRepository interface {
	Create(ctx context.Context, session model.Session) (model.Session, error)
	ListByUserID(ctx context.Context, userID string) ([]model.Session, error)
	UpdateLastUsedAt(ctx context.Context, sessionID string, lastUsedAt sql.NullTime) error
	UpdateStatus(ctx context.Context, sessionID string, status string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
	FindByID(ctx context.Context, sessionID string) (model.Session, error)
}

type PostgresSessionRepository struct {
	db *sql.DB
}

func NewPostgresSessionRepository(db *sql.DB) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session model.Session) (model.Session, error) {
	query := `
		INSERT INTO sessions (id, user_id, device_name, ip_address, status, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, device_name, ip_address, status, created_at, last_used_at
	`

	var created model.Session
	err := r.db.QueryRowContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.DeviceName,
		session.IPAddress,
		session.Status,
		session.CreatedAt,
		session.LastUsedAt,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.DeviceName,
		&created.IPAddress,
		&created.Status,
		&created.CreatedAt,
		&created.LastUsedAt,
	)
	if err != nil {
		return model.Session{}, err
	}

	return created, nil
}

func (r *PostgresSessionRepository) ListByUserID(ctx context.Context, userID string) ([]model.Session, error) {
	query := `
		SELECT id, user_id, device_name, ip_address, status, created_at, last_used_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var session model.Session
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.DeviceName,
			&session.IPAddress,
			&session.Status,
			&session.CreatedAt,
			&session.LastUsedAt,
		); err != nil {
			return nil, err
		}

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (r *PostgresSessionRepository) UpdateLastUsedAt(ctx context.Context, sessionID string, lastUsedAt sql.NullTime) error {
	query := `UPDATE sessions SET last_used_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, lastUsedAt, sessionID)
	return err
}

func (r *PostgresSessionRepository) UpdateStatus(ctx context.Context, sessionID string, status string) error {
	query := `UPDATE sessions SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, sessionID)
	return err
}

func (r *PostgresSessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	query := `UPDATE sessions SET status = 'REVOKED' WHERE user_id = $1 AND status = 'ACTIVE'`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *PostgresSessionRepository) FindByID(ctx context.Context, sessionID string) (model.Session, error) {
	query := `
		SELECT id, user_id, device_name, ip_address, status, created_at, last_used_at
		FROM sessions
		WHERE id = $1
	`

	var session model.Session
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceName,
		&session.IPAddress,
		&session.Status,
		&session.CreatedAt,
		&session.LastUsedAt,
	)
	if err != nil {
		return model.Session{}, err
	}

	return session, nil
}

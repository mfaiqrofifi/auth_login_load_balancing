package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"

	"load_balancing_project_auth/internal/model"
)

var ErrDuplicateEmail = errors.New("email already exists")

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id string) (model.User, error)
	CreateUser(ctx context.Context, user model.User) (model.User, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	query := `
		SELECT id, email, password_hash, is_email_verified, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, strings.ToLower(strings.TrimSpace(email))).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.IsEmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	query := `
		SELECT id, email, password_hash, is_email_verified, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.IsEmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (r *PostgresUserRepository) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	query := `
		INSERT INTO users (id, email, password_hash, is_email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, password_hash, is_email_verified, created_at, updated_at
	`

	var createdUser model.User
	err := r.db.QueryRowContext(
		ctx,
		query,
		user.ID,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.HashedPassword,
		user.IsEmailVerified,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(
		&createdUser.ID,
		&createdUser.Email,
		&createdUser.HashedPassword,
		&createdUser.IsEmailVerified,
		&createdUser.CreatedAt,
		&createdUser.UpdatedAt,
	)
	if err != nil {
		var pqError *pq.Error
		if errors.As(err, &pqError) && pqError.Code == "23505" {
			return model.User{}, ErrDuplicateEmail
		}

		return model.User{}, err
	}

	return createdUser, nil
}

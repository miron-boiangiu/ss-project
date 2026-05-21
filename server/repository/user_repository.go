package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mqtt-streaming-server/domain"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (repo *UserRepository) Save(ctx context.Context, email, password string) error {
	_, err := repo.db.Exec(ctx,
		"INSERT INTO users (email, password, role) VALUES ($1, $2, 'user')",
		email, password)
	return err
}

func (repo *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := repo.db.QueryRow(ctx,
		"SELECT email, password, role FROM users WHERE email = $1", email).
		Scan(&user.Email, &user.Password, &user.Role)
	if err == pgx.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

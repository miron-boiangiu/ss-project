package domain

import "context"

type User struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role,omitempty"`
}

type UserRepository interface {
	Save(ctx context.Context, email, password string) error
	FindByEmail(ctx context.Context, email string) (*User, error)
}

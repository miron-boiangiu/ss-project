package domain

import "context"

type User struct {
	Email    string `json:"email" bson:"email"`
	Password string `json:"password,omitempty" bson:"password"`
	Role     string `json:"role,omitempty" bson:"role"`
}

type UserRepository interface {
	Save(ctx context.Context, email, password string) error
	FindByEmail(ctx context.Context, email string) (*User, error)
}

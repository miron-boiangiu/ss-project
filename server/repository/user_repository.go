package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/domain"
)

type UserRepository struct {
	db *mongo.Database
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{db: db}
}

func (repo *UserRepository) Save(ctx context.Context, email, password string) error {
	collection := repo.db.Collection("users")
	_, err := collection.InsertOne(ctx, domain.User{
		Email:    email,
		Password: password,
		Role:     "user",
	})
	return err
}

func (repo *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	collection := repo.db.Collection("users")
	var user domain.User
	err := collection.FindOne(ctx, map[string]string{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

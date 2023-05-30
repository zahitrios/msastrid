package services

import (
	"context"
	"ms-astrid/products/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	db mongo.Client
}

func NewUserService(db *mongo.Client) *UserService {
	return &UserService{
		db: *db,
	}
}

func (u *UserService) GetRoleByUser(ctx context.Context, email string) string {
	collection := u.db.Database("gaia").Collection("users")

	var (
		user models.User
		role = models.Guest
	)

	if err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user); err == nil {
		role = user.Role
	}

	return role.String()
}

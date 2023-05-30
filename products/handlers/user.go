package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/services"
)

type user struct {
	Handler
	db      *mongo.Client
	service services.UserService
}

func NewUserHanlder(db *mongo.Client) *user {
	userService := services.NewUserService(db)

	return &user{
		db:      db,
		service: *userService,
	}
}

func (u *user) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/users/role/{email}", u.getRoleByUser).Methods(http.MethodGet)
}

func (u *user) getRoleByUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	role := u.service.GetRoleByUser(context.TODO(), vars["email"])
	json.NewEncoder(w).Encode(DataResponse{Data: role})
}

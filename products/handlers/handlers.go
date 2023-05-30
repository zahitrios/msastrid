package handlers

import (
	"encoding/json"
	"errors"
	"ms-astrid/products/horrors"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Handler interface {
	RegisterRoutes(r *mux.Router)
}

type errorResponse struct {
	Err string `json:"error"`
}

type DataResponse struct {
	Data string `json:"data"`
}

func HanlderError(err error, w http.ResponseWriter, r *http.Request) {
	statusCode := http.StatusInternalServerError

	switch err {
	case primitive.ErrInvalidHex:
		err = horrors.NewBadRequestError(err.Error())
	case mongo.ErrNilDocument, mongo.ErrNoDocuments:
		err = horrors.NewNotFoundError("No resource found")
	case http.ErrMissingFile:
		err = horrors.NewNotFoundError(err.Error())
	}

	var badRequestError *horrors.BadRequestError
	if errors.As(err, &badRequestError) {
		statusCode = badRequestError.StatusCode
	}

	var notFoundError *horrors.NotFoundError
	if errors.As(err, &notFoundError) {
		statusCode = notFoundError.StatusCode
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{
		Err: err.Error(),
	})
}

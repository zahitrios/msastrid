package handlers

import (
	"context"
	"encoding/json"
	"ms-astrid/products/services"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
)

type logger struct {
	Handler
	service services.LoggerService
}

func NewLoggerHanlder(db *mongo.Client) *logger {
	loggerService := services.NewLoggerService(db)

	return &logger{
		service: *loggerService,
	}
}

func (h *logger) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/logs", h.getLogs).Methods(http.MethodGet)
}

func (h *logger) getLogs(w http.ResponseWriter, r *http.Request) {
	h.service.AddPaginator(r)

	paginator, err := h.service.GetLogs(context.TODO(), r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"ms-astrid/products/services"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
)

type exporter struct {
	Handler
	service services.ExporterService
}

func NewExporterHandler(db *mongo.Client) *exporter {
	exporterService := services.NewExporterService(db)

	return &exporter{
		service: *exporterService,
	}
}

func (e *exporter) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/export/{collection}", e.export).Methods(http.MethodGet)
}

func (e *exporter) export(w http.ResponseWriter, r *http.Request) {
	collection := mux.Vars(r)["collection"]
	data := e.service.GetData(context.TODO(), collection)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s.csv", collection))
	wr := csv.NewWriter(w)

	if err := wr.WriteAll(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

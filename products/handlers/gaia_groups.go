package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/services"
)

type gaiaGroups struct {
	Handler
	db      *mongo.Client
	service services.GaiaGroupsService
}

func NewGaiaGroupsHanlder(db *mongo.Client) *gaiaGroups {
	gaiaGroupsService := services.NewGaiaGroupsService(db)

	return &gaiaGroups{
		db:      db,
		service: *gaiaGroupsService,
	}
}

func (c *gaiaGroups) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/gaia-groups", c.getGaiaGroups).Methods(http.MethodGet)
	r.HandleFunc("/gaia-groups", c.calculatePrices).Methods(http.MethodPost)
	r.HandleFunc("/gaia-groups", c.deleteGaiaGroups).Methods(http.MethodDelete)
	r.HandleFunc("/gaia-groups/publish", c.getGaiaGroupsToPublish).Methods(http.MethodGet)
	r.HandleFunc("/gaia-groups/publish", c.publishGaiaGroups).Methods(http.MethodPost)
	r.HandleFunc("/gaia-groups/{sku}", c.getGaiaGroup).Methods(http.MethodGet)
	r.HandleFunc("/gaia-groups/sync", c.syncGaiaGroups).Methods(http.MethodPost)
}

func (g *gaiaGroups) calculatePrices(w http.ResponseWriter, r *http.Request) {
	g.service.CalculatePrices(context.TODO())
	w.WriteHeader(http.StatusAccepted)
}

func (g *gaiaGroups) deleteGaiaGroups(w http.ResponseWriter, r *http.Request) {
	if err := g.service.DeleteGaiaGroups(context.TODO()); err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (g *gaiaGroups) getGaiaGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sku, err := g.service.GetGaiaGroup(context.TODO(), vars["sku"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(sku)
}

func (g *gaiaGroups) getGaiaGroups(w http.ResponseWriter, r *http.Request) {
	g.service.AddPaginator(r)
	g.service.AddSearch(r, "parent_sku")

	paginator, err := g.service.GetGaiaGroups(context.TODO(), g.service.GetFilter(), r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (g *gaiaGroups) getGaiaGroupsToPublish(w http.ResponseWriter, r *http.Request) {
	g.service.AddPaginator(r)
	g.service.AddSearch(r, "parent_sku")

	filter := g.service.GetFilter()
	filter["published"] = false

	paginator, err := g.service.GetGaiaGroups(context.TODO(), filter, r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (g *gaiaGroups) publishGaiaGroups(w http.ResponseWriter, r *http.Request) {
	if err := g.service.PublishGaiaGroups(context.TODO()); err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (g *gaiaGroups) syncGaiaGroups(w http.ResponseWriter, r *http.Request) {
	if err := g.service.SyncGaiaGroups(context.TODO()); err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

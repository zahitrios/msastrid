package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/models"
	"ms-astrid/products/services"
)

type campaign struct {
	Handler
	db      *mongo.Client
	service services.CampaignService
}

func NewCampaignHanlder(db *mongo.Client) *campaign {
	campaignService := services.NewCampaignService(db)

	return &campaign{
		db:      db,
		service: *campaignService,
	}
}

func (c *campaign) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/campaigns", c.getCampaigns).Methods(http.MethodGet)
	r.HandleFunc("/campaigns", c.createCampaign).Methods(http.MethodPost)
	r.HandleFunc("/campaigns/{id}", c.getCampaign).Methods(http.MethodGet)
	r.HandleFunc("/campaigns/{id}", c.updateCampaign).Methods(http.MethodPost)
	r.HandleFunc("/campaigns/delete/{id}", c.deleteCampaign).Methods(http.MethodPost)
}

func (c *campaign) getCampaigns(w http.ResponseWriter, r *http.Request) {
	c.service.AddPaginator(r)
	c.service.AddSearch(r, "name")

	paginator, err := c.service.GetCampaigns(context.TODO(), r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (c *campaign) createCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign models.Campaign
	json.NewDecoder(r.Body).Decode(&campaign)

	campaign, err := c.service.CreateCampaign(context.TODO(), campaign)
	if err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(campaign)
}

func (c *campaign) getCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaign, err := c.service.GetCampaign(context.TODO(), vars["id"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(campaign)
}

func (c *campaign) updateCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaign, err := c.service.GetCampaign(context.TODO(), vars["id"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	var campaignToUpdate models.Campaign
	json.NewDecoder(r.Body).Decode(&campaignToUpdate)
	campaignToUpdate.Id = campaign.Id
	campaignToUpdate.CreateAt = campaign.CreateAt

	if err := c.service.UpdateCampaign(context.TODO(), &campaignToUpdate); err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(campaignToUpdate)
}

func (c *campaign) deleteCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaign, err := c.service.GetCampaign(context.TODO(), vars["id"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	if campaign.Priority == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{
			Err: "Can not delete campaign with priority 0",
		})
		return
	}

	if err := c.service.DeleteCampaign(context.TODO(), campaign); err != nil {
		HanlderError(err, w, r)
		return
	}

	campaignItemsService := services.NewCampaignItemService(c.db)
	if err := campaignItemsService.DeleteCampaignItems(context.TODO(), campaign); err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

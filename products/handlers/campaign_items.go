package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/models"
	"ms-astrid/products/services"
)

type campaignItems struct {
	Handler
	db              *mongo.Client
	campaignService services.CampaignService
	service         services.CampaignItemService
}

func NewCampaignItemsHanlder(db *mongo.Client) *campaignItems {
	campaignService := services.NewCampaignService(db)
	campaignItemService := services.NewCampaignItemService(db)

	return &campaignItems{
		db:              db,
		campaignService: *campaignService,
		service:         *campaignItemService,
	}
}

func (ci *campaignItems) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/campaigns/base/items/bulk-approve-costs", ci.approveCostsInBaseCampaign).Methods(http.MethodPost)
	r.HandleFunc("/campaigns/base/items/pendding-costs", ci.getCampaignItemsWithCostUpdated).Methods(http.MethodGet)
	r.HandleFunc("/campaigns/{campaignId}/items", ci.getCampaignItemsByApproved).Methods(http.MethodGet)
	r.HandleFunc("/campaigns/{campaignId}/items/pendding", ci.getCampaignItemsByPendding).Methods(http.MethodGet)
	r.HandleFunc("/campaigns/{campaignId}/items/pendding/detail", ci.getDetailOfPendingItems).Methods(http.MethodGet)
	r.HandleFunc("/campaigns/{campaignId}/items/bulk-approve", ci.approveCampaignItems).Methods(http.MethodPost)
}

func (ci *campaignItems) approveCampaignItems(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	campaign, err := ci.campaignService.GetCampaign(context.TODO(), vars["campaignId"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	var skuList models.SkuList
	json.NewDecoder(r.Body).Decode(&skuList)

	if err := ci.service.ApproveCampaignItems(context.TODO(), campaign, skuList); err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(skuList)
}

func (ci *campaignItems) approveCostsInBaseCampaign(w http.ResponseWriter, r *http.Request) {
	campaign, err := ci.campaignService.GetBaseCampaign(context.TODO())

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	var skuList models.SkuList
	json.NewDecoder(r.Body).Decode(&skuList)

	if err := ci.service.ApproveCostsInBaseCampaign(context.TODO(), campaign, skuList); err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(skuList)
}

func (ci *campaignItems) getCampaignItemsByApproved(w http.ResponseWriter, r *http.Request) {
	ci.service.AddPaginator(r)

	vars := mux.Vars(r)
	campaign, err := ci.campaignService.GetCampaign(context.TODO(), vars["campaignId"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	paginator, err := ci.service.GetCampaignItemsByStatus(context.TODO(), campaign, models.ApprovedStatus, r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (ci *campaignItems) getCampaignItemsByPendding(w http.ResponseWriter, r *http.Request) {
	ci.service.AddPaginator(r)

	vars := mux.Vars(r)
	campaign, err := ci.campaignService.GetCampaign(context.TODO(), vars["campaignId"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	paginator, err := ci.service.GetCampaignItemsByStatus(context.TODO(), campaign, models.PenddingStatus, r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (ci *campaignItems) getCampaignItemsWithCostUpdated(w http.ResponseWriter, r *http.Request) {
	campaign, err := ci.campaignService.GetBaseCampaign(context.TODO())

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	ci.service.AddPaginator(r)
	ci.service.AddFilter("new_cost", bson.M{
		"$gt": 0,
	})
	paginator, err := ci.service.GetCampaignItemsByStatus(context.TODO(), campaign, "", r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (ci *campaignItems) getDetailOfPendingItems(w http.ResponseWriter, r *http.Request) {
	ci.service.AddPaginator(r)

	vars := mux.Vars(r)
	campaign, err := ci.campaignService.GetCampaign(context.TODO(), vars["campaignId"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	detailOfPendingItems := ci.service.GetDetailOfPendingItems(context.TODO(), campaign)

	json.NewEncoder(w).Encode(detailOfPendingItems)
}

package handlers

import (
	"context"
	"encoding/json"
	"log"
	"ms-astrid/products/services"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MergePriceHandler struct {
	Handler
	mergePriceService *services.MergePriceService
	priceListService  *services.PriceListService
	gaiaGroupService  *services.GaiaGroupsService
}

// getPriceList Handle get price list merged of "astrid" and "algolia"
func (handler *MergePriceHandler) getPriceList(writer http.ResponseWriter, request *http.Request) {
	handler.mergePriceService.AddPaginator(request)
	simpleItems, _ := handler.priceListService.GetAll(context.TODO(), bson.M{})
	groupedItems, _ := handler.gaiaGroupService.GetAll(context.TODO(), bson.M{})
	result := handler.mergePriceService.GetPriceList(simpleItems, groupedItems)

	errEncode := json.NewEncoder(writer).Encode(result)
	if errEncode != nil {
		log.Printf(errEncode.Error())
	}
}

// RegisterRoutes list of routes for handler
func (handler *MergePriceHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/merge-price-report", handler.getPriceList).Methods(http.MethodGet)
}

// NewMergePriceHandler factory handler
func NewMergePriceHandler(client *mongo.Client) *MergePriceHandler {
	mergePriceService := services.NewMergePriceService()
	return &MergePriceHandler{
		mergePriceService: mergePriceService,
		priceListService:  services.NewPriceListService(client),
		gaiaGroupService:  services.NewGaiaGroupsService(client),
	}
}

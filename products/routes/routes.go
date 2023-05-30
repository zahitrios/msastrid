package routes

import (
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/handlers"
)

func RegisterRoutes(client *mongo.Client, r *mux.Router) {
	handlerList := []any{
		handlers.NewCampaignHanlder(client),
		handlers.NewCampaignItemsHanlder(client),
		handlers.NewPriceListHanlder(client),
		handlers.NewGaiaGroupsHanlder(client),
		handlers.NewMergePriceHandler(client),
		handlers.NewUserHanlder(client),
		handlers.NewExporterHandler(client),
		handlers.NewLoggerHanlder(client),
	}

	for _, handler := range handlerList {
		if h, ok := handler.(handlers.Handler); ok {
			h.RegisterRoutes(r)
		}
	}
}

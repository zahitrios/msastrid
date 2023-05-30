package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/services"
)

type priceList struct {
	Handler
	db      *mongo.Client
	service services.PriceListService
}

func NewPriceListHanlder(db *mongo.Client) *priceList {
	priceListService := services.NewPriceListService(db)

	return &priceList{
		db:      db,
		service: *priceListService,
	}
}

func (p *priceList) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/price-list", p.getPriceList).Methods(http.MethodGet)
	r.HandleFunc("/price-list/publish", p.getPriceListToPublish).Methods(http.MethodGet)
	r.HandleFunc("/price-list/{sku}", p.getSku).Methods(http.MethodGet)
	r.HandleFunc("/price-list", p.createPriceList).Methods(http.MethodPost)
	r.HandleFunc("/price-list/publish", p.publishPriceList).Methods(http.MethodPost)
}

func (p *priceList) getPriceList(w http.ResponseWriter, r *http.Request) {
	p.service.AddPaginator(r)
	p.service.AddSearch(r, "sku")

	paginator, err := p.service.GetPriceList(context.TODO(), p.service.GetFilter(), r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(paginator)
}

func (p *priceList) getPriceListToPublish(w http.ResponseWriter, r *http.Request) {
	p.service.AddPaginator(r)
	p.service.AddSearch(r, "sku")

	filter := p.service.GetFilter()
	filter["published"] = false

	priceList, err := p.service.GetPriceList(context.TODO(), filter, r)

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(priceList)
}

func (p *priceList) getSku(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sku, err := p.service.GetSku(context.TODO(), vars["sku"])

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(sku)
}

func (p *priceList) createPriceList(w http.ResponseWriter, r *http.Request) {
	err := p.service.CreatePriceList(context.TODO())

	if err != nil {
		HanlderError(err, w, r)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (p *priceList) publishPriceList(w http.ResponseWriter, r *http.Request) {
	err := p.service.PublishPriceList(context.TODO())

	if err != nil {
		HanlderError(err, w, r)
	}

	w.WriteHeader(http.StatusAccepted)
}

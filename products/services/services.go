package services

import (
	"ms-astrid/products/paginator"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BaseService struct {
	filter bson.M
	opts   *options.FindOptions
}

func (b *BaseService) AddPaginator(r *http.Request) {
	b.opts = paginator.NewOptions(r)
	b.filter = bson.M{}
}

func (b *BaseService) AddFilter(key string, value any) {
	if b.filter == nil {
		b.filter = bson.M{}
	}

	b.filter[key] = value
}

func (b *BaseService) AddSearch(r *http.Request, search string) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	if b.filter == nil || query == "" {
		b.filter = bson.M{}
	}

	if query != "" {
		b.filter[search] = bson.M{"$regex": query, "$options": "i"}
	}
}

func (b *BaseService) GetFilter() bson.M {
	return b.filter
}

package paginator

import (
	"math"
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultLimit = 100
	defaultPage  = 1
)

type Paginator struct {
	Total   int64         `json:"total"`
	Pages   int64         `json:"pages"`
	Results int64         `json:"results"`
	Data    []interface{} `json:"data"`
}

func NewOptions(r *http.Request) *options.FindOptions {
	query := r.URL.Query()

	limitParam := query.Get("limit")
	limit, _ := strconv.ParseInt(limitParam, 10, 0)
	if limit == 0 {
		limit = defaultLimit
	}

	pageParam := query.Get("page")
	page, _ := strconv.ParseInt(pageParam, 10, 0)
	if page == 0 {
		page = defaultPage
	}

	skip := (page * limit) - limit

	return &options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
	}
}

func NewPaginator(data []interface{}, total int64, r *http.Request) *Paginator {
	results := int64(len(data))
	pages := int64(0)
	query := r.URL.Query()

	limitParam := query.Get("limit")
	limit, _ := strconv.ParseInt(limitParam, 10, 0)
	if limit == 0 {
		limit = defaultLimit
	}

	if total > 0 && limit > 0 {
		pages = int64(math.Ceil(float64(total) / float64(limit)))
	}

	return &Paginator{
		Total:   total,
		Pages:   pages,
		Results: results,
		Data:    data,
	}
}

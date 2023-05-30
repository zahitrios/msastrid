package services

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/models"
	"ms-astrid/products/paginator"
	"ms-astrid/products/utils"
)

type LoggerService struct {
	BaseService
	db mongo.Client
}

func NewLoggerService(db *mongo.Client) *LoggerService {
	return &LoggerService{
		db: *db,
	}
}

func (s *LoggerService) GetLogs(ctx context.Context, r *http.Request) (*paginator.Paginator, error) {
	collection := s.db.Database("gaia").Collection("logs")
	total, _ := collection.CountDocuments(ctx, s.filter)
	s.opts.Sort = bson.M{"_id": -1}
	cursor, err := collection.Find(ctx, s.filter, s.opts)

	if err != nil {
		return nil, err
	}

	var logs []interface{}
	loc, _ := time.LoadLocation("America/Mexico_City")

	for cursor.Next(ctx) {
		var logger models.Logger
		cursor.Decode(&logger)
		logger.CreateAt = logger.CreateAt.In(loc)
		logs = append(logs, logger)
	}

	return paginator.NewPaginator(logs, total, r), nil

}

func (s *LoggerService) LogRequest(w *utils.ResponseWriter, r *http.Request) {
	logger := models.Logger{
		Method:     r.Method,
		Url:        r.URL.RequestURI(),
		UserAgent:  r.UserAgent(),
		Body:       string(w.ReqBodyBytes),
		Ip:         r.Header.Get("X-Forwarded-For"),
		Email:      r.URL.Query().Get("email"),
		StatusCode: w.GetStatusCode(),
		Response:   string(w.ResBodyBytes),
		CreateAt:   time.Now(),
	}

	s.db.Database("gaia").Collection("logs").InsertOne(r.Context(), logger)
}

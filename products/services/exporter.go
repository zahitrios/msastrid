package services

import (
	"context"
	"fmt"
	"ms-astrid/products/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ExporterService struct {
	db   mongo.Client
	data [][]string
}

func NewExporterService(db *mongo.Client) *ExporterService {
	exporterService := &ExporterService{
		db: *db,
	}

	return exporterService
}

func (s *ExporterService) GetData(ctx context.Context, exporter string) [][]string {
	s.data = [][]string{{}}

	if exporter == "price-list" {
		s.createPriceList(ctx)
	} else if exporter == "gaia-groups" {
		s.createGaiaGroups(ctx)
	} else if exporter == "logs" {
		s.createLogs(ctx)
	}

	return s.data
}

func (s *ExporterService) createGaiaGroups(ctx context.Context) {
	collection := s.db.Database("gaia").Collection("gaia_groups")
	cursor, _ := collection.Find(ctx, bson.M{})

	s.data = [][]string{{
		"sku",
		"price",
		"msrp",
		"published",
	}}

	for cursor.Next(ctx) {
		var gaiaGroup models.GaiaGroup
		cursor.Decode(&gaiaGroup)

		s.data = append(s.data, []string{
			gaiaGroup.ParentSku,
			fmt.Sprintf("%.2f", gaiaGroup.Price),
			fmt.Sprintf("%.2f", gaiaGroup.Msrp),
			fmt.Sprintf("%v", gaiaGroup.Published),
		})
	}
}

func (s *ExporterService) createLogs(ctx context.Context) {
	collection := s.db.Database("gaia").Collection("logs")
	cursor, _ := collection.Find(ctx, bson.M{})

	s.data = [][]string{{
		"method",
		"url",
		"user_agent",
		"body",
		"ip",
		"email",
		"created_at",
	}}

	loc, _ := time.LoadLocation("America/Mexico_City")

	for cursor.Next(ctx) {
		var logger models.Logger
		cursor.Decode(&logger)

		logger.CreateAt = logger.CreateAt.In(loc)

		s.data = append(s.data, []string{
			logger.Method,
			logger.Url,
			logger.UserAgent,
			logger.Body,
			logger.Ip,
			logger.Email,
			logger.CreateAt.String(),
		})
	}
}

func (s *ExporterService) createPriceList(ctx context.Context) {
	collection := s.db.Database("gaia").Collection("price_list")
	cursor, _ := collection.Find(ctx, bson.M{})

	s.data = [][]string{{
		"sku",
		"price",
		"msrp",
		"priority",
		"published",
	}}

	for cursor.Next(ctx) {
		var sku models.Sku
		cursor.Decode(&sku)

		s.data = append(s.data, []string{
			sku.Sku,
			fmt.Sprintf("%.2f", sku.Price),
			fmt.Sprintf("%.2f", sku.Msrp),
			fmt.Sprintf("%d", sku.Priority),
			fmt.Sprintf("%v", sku.Published),
		})
	}
}

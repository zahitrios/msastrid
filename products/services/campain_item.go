package services

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/horrors"
	"ms-astrid/products/models"
	"ms-astrid/products/paginator"
	"ms-astrid/products/utils"
)

const (
	skuIndex   = 0
	priceIndex = 1
	costIndex  = 2
	msrpIndex  = 3

	batchProcess = 5000
)

type CampaignItemService struct {
	BaseService
	db mongo.Client
}

func NewCampaignItemService(db *mongo.Client) *CampaignItemService {
	return &CampaignItemService{
		db: *db,
	}
}

func (s *CampaignItemService) ApproveCampaignItems(ctx context.Context, campaign models.Campaign, skuList models.SkuList) error {
	if len(skuList.Skus) == 0 {
		return horrors.NewBadRequestError(utils.Translate("no_skus_to_update"))
	}

	collection := s.db.Database("gaia").Collection("campaign_items")
	filter := bson.M{
		"campaign": campaign.Id,
		"sku": bson.M{
			"$in": skuList.Skus,
		},
	}

	update := bson.M{
		"$set": bson.M{
			"status": models.ApprovedStatus,
		},
	}

	if _, err := collection.UpdateMany(ctx, filter, update); err != nil {
		return err
	}

	return nil
}

func (s *CampaignItemService) ApproveCostsInBaseCampaign(ctx context.Context, campaign models.Campaign, skuList models.SkuList) error {
	if len(skuList.Skus) == 0 {
		return horrors.NewBadRequestError(utils.Translate("no_skus_to_update"))
	}

	if campaign.Type != models.BaseCampaign {
		return horrors.NewBadRequestError(utils.Translate("campaign_type_is_not_valid"))
	}

	filter := bson.M{
		"campaign": campaign.Id,
		"sku": bson.M{
			"$in": skuList.Skus,
		},
	}

	update := bson.M{
		"$rename": bson.M{
			"cost": "old_cost",
		},
	}

	collection := s.db.Database("gaia").Collection("campaign_items")

	if _, err := collection.UpdateMany(ctx, filter, update); err != nil {
		return err
	}

	update = bson.M{
		"$rename": bson.M{
			"new_cost": "cost",
		},
	}

	if _, err := collection.UpdateMany(ctx, filter, update); err != nil {
		return err
	}

	s.recalculateCampaignItems(ctx, campaign, skuList.Skus)

	return nil
}

func (s *CampaignItemService) DeleteCampaignItems(ctx context.Context, campaign models.Campaign) error {
	collection := s.db.Database("gaia").Collection("campaign_items")

	filter := bson.M{
		"campaign": campaign.Id,
	}

	if _, err := collection.DeleteMany(ctx, filter); err != nil {
		return err
	}

	return nil
}

func (s *CampaignItemService) GetCampaignItemsByStatus(ctx context.Context, campaign models.Campaign, status string, r *http.Request) (*paginator.Paginator, error) {
	if s.filter == nil {
		s.filter = bson.M{}
	}

	s.filter["campaign"] = campaign.Id

	if status != "" {
		s.filter["status"] = status
	}

	s.opts.SetSort(bson.M{
		"profit": 1,
	})

	collection := s.db.Database("gaia").Collection("campaign_items")
	total, _ := collection.CountDocuments(ctx, s.filter)
	cursor, err := collection.Find(ctx, s.filter, s.opts)

	if err != nil {
		return nil, err
	}

	var campaignItems []interface{}

	for cursor.Next(ctx) {
		var campaignItem models.CampaignItem
		cursor.Decode(&campaignItem)
		campaignItems = append(campaignItems, campaignItem)
	}

	return paginator.NewPaginator(campaignItems, total, r), nil
}

func (s *CampaignItemService) GetDetailOfPendingItems(ctx context.Context, campaign models.Campaign) models.DetailOfPendingItems {
	filter := bson.M{
		"campaign": campaign.Id,
		"status":   models.PenddingStatus,
		"profit": bson.M{
			"$lte": 30,
			"$gt":  0,
		},
	}
	detailOfPendingItems := models.DetailOfPendingItems{}
	detailOfPendingItems.TotalUnder30, _ = s.db.Database("gaia").Collection("campaign_items").CountDocuments(ctx, filter)
	filter["profit"] = bson.M{"$lte": 0}
	detailOfPendingItems.TotalWithNegativeOrZero, _ = s.db.Database("gaia").Collection("campaign_items").CountDocuments(ctx, filter)
	return detailOfPendingItems
}

func (s *CampaignItemService) GetTotalGroupedPendingItems(ctx context.Context, campaignIds []primitive.ObjectID) map[primitive.ObjectID]int {
	match := bson.M{
		"status": models.PenddingStatus,
	}

	if campaignIds != nil {
		match["campaign"] = bson.M{
			"$in": campaignIds,
		}
	}

	pipe := []bson.M{
		{
			"$match": match,
		}, {
			"$group": bson.M{
				"_id": "$campaign",
				"total_pending_items": bson.M{
					"$sum": 1,
				},
			},
		},
	}

	c := s.db.Database("gaia").Collection("campaign_items")
	cursor, _ := c.Aggregate(ctx, pipe)

	data := make(map[primitive.ObjectID]int)

	for cursor.Next(ctx) {
		var pendingItems models.PendingItems
		cursor.Decode(&pendingItems)
		data[pendingItems.Campaign] = pendingItems.Total
	}

	return data
}

func (s *CampaignItemService) NewCampaignItems(ctx context.Context, campaign models.Campaign, reader csv.Reader) error {
	var (
		count      int
		operations []interface{}
	)

	rows := s.getRowsFromReader(reader)
	newSkus := s.getSkusFromRows(rows)
	baseItems, _ := s.getBaseItemsBySkus(ctx, newSkus)

	if s.isTemporal(campaign) {
		if exists, skusNotFound := s.existsSkusInBaseCampaign(newSkus, baseItems); !exists {
			return fmt.Errorf(utils.Translate("skus_not_found_in_base"), skusNotFound)
		}
	}

	collection := s.db.Database("gaia").Collection("campaign_items")

	for _, row := range rows {
		count++

		if s.isTemporal(campaign) {
			campaignItemBase := s.getCampaignItemInBaseBySku(row[skuIndex], baseItems)
			row[costIndex] = fmt.Sprintf("%f", campaignItemBase.Cost)
		}

		campaignItem, err := s.newCampaignItem(row, campaign)

		if err != nil {
			continue
		}

		operations = append(operations, campaignItem)

		if (count % batchProcess) == 0 {
			if _, err = collection.InsertMany(ctx, operations); err != nil {
				return err
			}
			operations = nil
		}
	}

	var errInsert error

	if len(operations) > 0 {
		_, errInsert = collection.InsertMany(ctx, operations)
	}

	return errInsert
}

func (s *CampaignItemService) UpdateCampaignItems(ctx context.Context, campaign models.Campaign, reader csv.Reader) error {
	var (
		count      int
		operations []mongo.WriteModel
	)

	rows := s.getRowsFromReader(reader)
	newSkus := s.getSkusFromRows(rows)
	baseItems, _ := s.getBaseItemsBySkus(ctx, newSkus)

	if s.isTemporal(campaign) {
		if exists, skusNotFound := s.existsSkusInBaseCampaign(newSkus, baseItems); !exists {
			return fmt.Errorf(utils.Translate("skus_not_found_in_base"), skusNotFound)
		}
	}

	collection := s.db.Database("gaia").Collection("campaign_items")

	for _, row := range rows {
		count++

		newCost, _ := strconv.ParseFloat(row[costIndex], 64)

		campaignItemBase := s.getCampaignItemInBaseBySku(row[skuIndex], baseItems)
		row[costIndex] = fmt.Sprintf("%f", campaignItemBase.Cost)

		campaignItem, err := s.newCampaignItem(row, campaign)

		if err != nil {
			continue
		}

		// siempre el item va tener el costo de la base
		campaignItem.Cost = campaignItemBase.Cost

		if !s.isTemporal(campaign) {
			if campaignItem.Price == campaignItemBase.Price {
				campaignItem.Status = models.ApprovedStatus
			}

			if newCost != campaignItem.Cost {
				campaignItem.NewCost = newCost
			} else {
				campaignItem.NewCost = campaignItemBase.NewCost
			}
		}

		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(bson.M{
			"campaign": campaignItem.Campaign,
			"sku":      campaignItem.Sku,
		})
		operation.SetUpdate(bson.M{"$set": campaignItem})
		operation.SetUpsert(true)

		operations = append(operations, operation)

		if (count % batchProcess) != 0 {
			if _, err = collection.BulkWrite(ctx, operations, nil); err != nil {
				return err
			}

			var cleanOperations []mongo.WriteModel
			operations = cleanOperations
		}
	}

	var errUpdate error

	if len(operations) > 0 {
		_, errUpdate = collection.BulkWrite(ctx, operations, nil)
	}

	return errUpdate
}

func (s *CampaignItemService) existsSkusInBaseCampaign(newSkus []string, baseItems map[string]models.CampaignItem) (bool, string) {
	skusNotFound := []string{}
	exists := true

	for _, newSku := range newSkus {
		if _, ok := baseItems[newSku]; ok {
			continue
		}

		exists = false
		skusNotFound = append(skusNotFound, newSku)
	}

	return exists, strings.Join(skusNotFound, ",")
}

func (s *CampaignItemService) getBaseItemsBySkus(ctx context.Context, skus []string) (map[string]models.CampaignItem, error) {
	campaignService := NewCampaignService(&s.db)
	baseCampaign, err := campaignService.GetBaseCampaign(ctx)

	skusGrouped := map[string]models.CampaignItem{}

	if err != nil {
		return skusGrouped, err
	}

	cursor, err := s.db.Database("gaia").Collection("campaign_items").Find(ctx, bson.M{
		"campaign": baseCampaign.Id,
		"sku": bson.M{
			"$in": skus,
		},
	})

	if err != nil {
		return skusGrouped, err
	}

	for cursor.Next(ctx) {
		var sku models.CampaignItem
		cursor.Decode(&sku)
		skusGrouped[sku.Sku] = sku
	}

	return skusGrouped, nil
}

func (s *CampaignItemService) getCampaignItemInBaseBySku(sku string, baseItems map[string]models.CampaignItem) models.CampaignItem {
	var campaignItemFound models.CampaignItem

	if baseItem, ok := baseItems[sku]; ok {
		campaignItemFound = baseItem
	}

	return campaignItemFound
}

func (s *CampaignItemService) getRowsFromReader(reader csv.Reader) [][]string {
	rows := [][]string{}

	for {
		row, err := reader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			continue
		}

		rows = append(rows, row)
	}

	return rows
}

func (s *CampaignItemService) getSkusFromRows(campaignItems [][]string) []string {
	var skus []string

	for _, campaignItem := range campaignItems {
		skus = append(skus, campaignItem[skuIndex])
	}

	return skus
}

func (s *CampaignItemService) isTemporal(campaign models.Campaign) bool {
	return campaign.Type == models.TemporaryCampaign
}

func (s *CampaignItemService) newCampaignItem(row []string, campaign models.Campaign) (*models.CampaignItem, error) {
	if len(row) <= 1 {
		return nil, errors.New("row not valid")
	}

	cost, err := strconv.ParseFloat(row[costIndex], 64)
	if err != nil {
		return nil, err
	}

	msrp, _ := strconv.ParseFloat(row[msrpIndex], 64)

	price, err := strconv.ParseFloat(row[priceIndex], 64)
	if err != nil {
		return nil, err
	}

	sku := row[skuIndex]

	return models.NewCampaignItem(campaign, sku, cost, msrp, price), nil
}

func (s *CampaignItemService) recalculateCampaignItems(ctx context.Context, campaign models.Campaign, skus []string) {
	filter := bson.M{
		"campaign": campaign.Id,
		"sku": bson.M{
			"$in": skus,
		},
	}

	collection := s.db.Database("gaia").Collection("campaign_items")
	cursor, _ := collection.Find(ctx, filter)

	var operations []mongo.WriteModel

	for cursor.Next(ctx) {
		var campaignItem models.CampaignItem
		cursor.Decode(&campaignItem)

		campaignItemRecalculated := models.NewCampaignItem(campaign, campaignItem.Sku, campaignItem.Cost, campaignItem.Msrp, campaignItem.Price)

		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(bson.M{
			"campaign": campaignItemRecalculated.Campaign,
			"sku":      campaignItemRecalculated.Sku,
		})
		operation.SetUpdate(bson.M{"$set": campaignItemRecalculated})
		operations = append(operations, operation)
	}

	collection.BulkWrite(ctx, operations, nil)
}

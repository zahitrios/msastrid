package services

import (
	"context"
	"log"
	"net/http"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/horrors"
	"ms-astrid/products/models"
	"ms-astrid/products/paginator"
	"ms-astrid/products/queues"
	"ms-astrid/products/utils"
)

type PriceListService struct {
	BaseService
	db              mongo.Client
	campaignService CampaignService
	priorities      []int
	priceList       []interface{}
	skusGrouped     map[int]map[string]models.CampaignItem
	someoneChanged  bool
}

func NewPriceListService(db *mongo.Client) *PriceListService {
	campaignService := NewCampaignService(db)

	return &PriceListService{
		db:              *db,
		campaignService: *campaignService,
	}
}

func (p *PriceListService) CreatePriceList(ctx context.Context) error {
	log.Println("starting price list")

	baseCampaign, err := p.campaignService.GetBaseCampaign(ctx)
	if err != nil {
		return horrors.NewBadRequestError(utils.Translate("base_campaign_inactive"))
	}

	cursor, err := p.getApprovedItemsByCampaign(ctx, baseCampaign)
	if err != nil {
		return horrors.NewBadRequestError(utils.Translate("base_campaign_no_skus"))
	}

	campaigns := p.getActiveCampaigns(ctx)
	p.skusGrouped = p.getSkusGrouped(ctx, campaigns)
	p.orderPriorities()
	p.priceList = nil

	lastPriceGrouped := p.getLastPriceGrouped(ctx)

	p.db.Database("gaia").Collection("campaigns").UpdateMany(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"applied": false}},
	)

	for cursor.Next(ctx) {
		var campaignItem models.CampaignItem
		cursor.Decode(&campaignItem)
		sku := p.getSkuWithHigherPriority(campaignItem)

		if lastSku, ok := lastPriceGrouped[sku.Sku]; ok {
			sku.Published = lastSku.Published

			if lastSku.Price != sku.Price || lastSku.Msrp != sku.Msrp {
				sku.Published = false
			}
		}

		if !sku.Published {
			p.someoneChanged = true
		}

		p.priceList = append(p.priceList, *sku)
	}

	p.deactivateInactiveCampaigns(ctx)
	p.setAppliedToCampaigns(ctx, campaigns)

	if !p.someoneChanged {
		return nil
	}

	if _, err := p.db.Database("gaia").Collection("price_list").DeleteMany(ctx, bson.M{}); err != nil {
		log.Println("error cleaning price list: ", err)
		return err
	}

	_, err = p.db.Database("gaia").Collection("price_list").InsertMany(ctx, p.priceList)

	return err
}

func (p *PriceListService) GetPriceList(ctx context.Context, filter bson.M, r *http.Request) (*paginator.Paginator, error) {
	collection := p.db.Database("gaia").Collection("price_list")
	total, _ := collection.CountDocuments(ctx, filter)
	cursor, err := collection.Find(ctx, filter, p.opts)

	if err != nil {
		return nil, err
	}

	var skus []interface{}

	for cursor.Next(ctx) {
		var sku models.Sku
		cursor.Decode(&sku)
		skus = append(skus, sku)
	}

	return paginator.NewPaginator(skus, total, r), nil
}

func (p *PriceListService) GetAll(ctx context.Context, filter bson.M) ([]models.Sku, error) {
	var skus []models.Sku
	collection := p.db.Database("gaia").Collection("price_list")
	cursor, err := collection.Find(ctx, filter, p.opts)

	if err == nil {
		err = cursor.All(ctx, &skus)
	}

	return skus, err
}

func (p *PriceListService) GetSku(ctx context.Context, sku string) (models.Sku, error) {
	collection := p.db.Database("gaia").Collection("price_list")

	var skuModel models.Sku
	err := collection.FindOne(ctx, bson.M{"sku": sku}).Decode(&skuModel)

	return skuModel, err
}

func (p *PriceListService) PublishPriceList(ctx context.Context) error {
	q, err := queues.NewPublisher()

	if err != nil {
		return err
	}

	collection := p.db.Database("gaia").Collection("price_list")
	cursor, err := collection.Find(ctx, bson.M{"published": false})
	if err != nil {
		return err
	}

	var (
		skusToPublish []interface{}
		skusToUpdate  []string
	)

	for cursor.Next(ctx) {
		var skuRabbit models.SkuRabbit
		cursor.Decode(&skuRabbit)
		skusToPublish = append(skusToPublish, skuRabbit)
		skusToUpdate = append(skusToUpdate, skuRabbit.Sku)
	}

	if len(skusToPublish) > 0 {
		q.PublishSkus(skusToPublish)
	}

	collection.UpdateMany(
		ctx,
		bson.M{"sku": bson.M{"$in": skusToUpdate}},
		bson.M{"$set": bson.M{"published": true}},
	)

	return nil
}

func (p *PriceListService) deactivateInactiveCampaigns(ctx context.Context) {
	filter := bson.M{
		"type": models.TemporaryCampaign,
		"end_date": bson.M{
			"$lt": time.Now(),
		},
	}

	update := bson.M{
		"$set": bson.M{
			"enable": false,
		},
	}

	p.db.Database("gaia").Collection("campaigns").UpdateMany(ctx, filter, update)
}

func (p *PriceListService) getActiveCampaigns(ctx context.Context) []models.Campaign {
	collection := p.db.Database("gaia").Collection("campaigns")

	now := time.Now()
	filter := bson.M{
		"enable": true,
		"type":   models.TemporaryCampaign,
		"start_date": bson.M{
			"$lte": now,
		},
		"end_date": bson.M{
			"$gte": now,
		},
	}

	cursor, _ := collection.Find(ctx, filter)
	var campaigns []models.Campaign

	for cursor.Next(ctx) {
		var campaign models.Campaign
		cursor.Decode(&campaign)
		campaigns = append(campaigns, campaign)
	}

	return campaigns
}

func (p *PriceListService) getApprovedItemsByCampaign(ctx context.Context, campaign models.Campaign) (*mongo.Cursor, error) {
	filter := bson.M{
		"campaign": campaign.Id,
		"status":   models.ApprovedStatus,
	}

	collection := p.db.Database("gaia").Collection("campaign_items")
	return collection.Find(ctx, filter)
}

func (p *PriceListService) getLastPriceGrouped(ctx context.Context) map[string]models.Sku {
	collection := p.db.Database("gaia").Collection("price_list")
	cursor, _ := collection.Find(ctx, bson.M{})

	lastListPriceGroup := map[string]models.Sku{}

	for cursor.Next(ctx) {
		var sku models.Sku
		cursor.Decode(&sku)
		lastListPriceGroup[sku.Sku] = sku
	}

	return lastListPriceGroup
}

func (p *PriceListService) getSkusGrouped(ctx context.Context, campaigns []models.Campaign) map[int]map[string]models.CampaignItem {
	grouped := map[int]map[string]models.CampaignItem{}

	var ids bson.A
	for _, campaign := range campaigns {
		ids = append(ids, campaign.Id)
	}

	if len(ids) == 0 {
		return grouped
	}

	pipe := []bson.M{
		{
			"$match": bson.M{
				"campaign": bson.M{
					"$in": ids,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$campaign",
				"priority": bson.M{
					"$first": "$priority",
				},
				"campaign_items": bson.M{
					"$push": bson.M{
						"sku":      "$sku",
						"price":    "$price",
						"msrp":     "$msrp",
						"cost":     "$cost",
						"status":   "$status",
						"priority": "$priority",
					},
				}},
		}}

	c := p.db.Database("gaia").Collection("campaign_items")
	cursor, _ := c.Aggregate(ctx, pipe)

	for cursor.Next(ctx) {
		var campaignItemGrouped models.CampaignItemGrouped
		cursor.Decode(&campaignItemGrouped)

		grouped[campaignItemGrouped.Priority] = map[string]models.CampaignItem{}

		for _, campaignItem := range campaignItemGrouped.CampaignItems {
			if campaignItem.Status != models.ApprovedStatus {
				continue
			}

			grouped[campaignItemGrouped.Priority][campaignItem.Sku] = campaignItem
		}
	}

	return grouped
}

func (p *PriceListService) getSkuWithHigherPriority(baseSku models.CampaignItem) *models.Sku {
	tmpSku := baseSku

	for _, priority := range p.priorities {
		if foundSku, ok := p.skusGrouped[priority][baseSku.Sku]; ok {
			tmpSku = foundSku
			break
		}
	}

	return &models.Sku{
		Sku:      tmpSku.Sku,
		Price:    tmpSku.Price,
		Msrp:     tmpSku.Msrp,
		Cost:     tmpSku.Cost,
		Priority: tmpSku.Priority,
	}
}

func (p *PriceListService) orderPriorities() {
	priorities := []int{}

	for key := range p.skusGrouped {
		priorities = append(priorities, key)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(priorities)))

	p.priorities = priorities
}

func (p *PriceListService) setAppliedToCampaigns(ctx context.Context, campaigns []models.Campaign) {
	var ids bson.A
	for _, campaign := range campaigns {
		ids = append(ids, campaign.Id)
	}

	p.db.Database("gaia").Collection("campaigns").UpdateMany(
		ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"applied": true}},
	)
}

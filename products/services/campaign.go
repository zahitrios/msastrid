package services

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/horrors"
	"ms-astrid/products/models"
	"ms-astrid/products/paginator"
	"ms-astrid/products/utils"
)

type CampaignService struct {
	BaseService
	db mongo.Client
}

func NewCampaignService(db *mongo.Client) *CampaignService {
	return &CampaignService{
		db: *db,
	}
}

func (cs *CampaignService) isPriorityValid(ctx context.Context, campaign models.Campaign) bool {
	filter := bson.M{
		"priority": campaign.Priority,
	}

	if !campaign.Id.IsZero() {
		filter["_id"] = bson.M{"$ne": campaign.Id}
	}

	results, _ := cs.db.Database("gaia").Collection("campaigns").CountDocuments(ctx, filter)

	return results == 0
}

func (cs *CampaignService) valid(ctx context.Context, campaign *models.Campaign) error {
	if err := campaign.Valid(); err != nil {
		return err
	}

	if campaign.Type == models.BaseCampaign {
		baseCampaign, err := cs.GetBaseCampaign(ctx)

		if baseCampaign.Id == campaign.Id {
			return nil
		}

		if err == nil {
			return horrors.NewBadRequestError(utils.Translate("base_campaign_exists"))
		}
	}

	if !cs.isPriorityValid(ctx, *campaign) {
		return horrors.NewBadRequestError(utils.Translate("temporary_campaign_same_priority"))
	}

	return nil
}

func (cs *CampaignService) GetBaseCampaign(ctx context.Context) (models.Campaign, error) {
	collection := cs.db.Database("gaia").Collection("campaigns")

	filter := bson.M{
		"enable": true,
		"type":   models.BaseCampaign,
	}

	var campaign models.Campaign
	if err := collection.FindOne(ctx, filter).Decode(&campaign); err != nil {
		return campaign, err
	}

	return campaign, nil
}

func (cs *CampaignService) GetCampaigns(ctx context.Context, r *http.Request) (*paginator.Paginator, error) {
	collection := cs.db.Database("gaia").Collection("campaigns")
	total, _ := collection.CountDocuments(ctx, cs.filter)
	cs.opts.Sort = bson.M{"_id": -1}
	cursor, err := collection.Find(ctx, cs.filter, cs.opts)

	if err != nil {
		return nil, err
	}

	var campaigns []interface{}

	campaignItemService := NewCampaignItemService(&cs.db)
	totalPendingItems := campaignItemService.GetTotalGroupedPendingItems(ctx, nil)

	for cursor.Next(ctx) {
		var campaign models.Campaign
		cursor.Decode(&campaign)
		campaign.ConvertDates()
		campaign.TotalPendingItems = totalPendingItems[campaign.Id]
		campaigns = append(campaigns, campaign)
	}

	return paginator.NewPaginator(campaigns, total, r), nil
}

func (cs *CampaignService) GetCampaign(ctx context.Context, id string) (models.Campaign, error) {
	var campaign models.Campaign
	_id, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return campaign, err
	}

	collection := cs.db.Database("gaia").Collection("campaigns")
	err = collection.FindOne(ctx, bson.M{"_id": _id}).Decode(&campaign)

	if err == mongo.ErrNoDocuments {
		return campaign, horrors.NewNotFoundError(utils.Translate("resource_not_found"))
	}

	campaign.ConvertDates()

	campaignIds := []primitive.ObjectID{_id}
	campaignItemService := NewCampaignItemService(&cs.db)
	campaign.TotalPendingItems = campaignItemService.GetTotalGroupedPendingItems(ctx, campaignIds)[_id]

	return campaign, err
}

func (cs *CampaignService) CreateCampaign(ctx context.Context, campaign models.Campaign) (models.Campaign, error) {
	if err := cs.valid(ctx, &campaign); err != nil {
		return campaign, err
	}

	reader, err := utils.GetReader(campaign.Filename)
	if err != nil {
		return campaign, err
	}

	campaign.Enable = false

	campaign.StartDate = campaign.StartDate.Add(time.Duration(+6) * time.Hour)
	campaign.EndDate = campaign.EndDate.Add(time.Duration(+6) * time.Hour)

	campaign.CreateAt = time.Now()
	campaign.UpdatedAt = campaign.CreateAt

	collection := cs.db.Database("gaia").Collection("campaigns")
	result, err := collection.InsertOne(ctx, campaign)
	campaign.ConvertDates()

	if err != nil {
		return campaign, err
	}

	campaign.Id = result.InsertedID.(primitive.ObjectID)

	campaignItemService := NewCampaignItemService(&cs.db)
	if err := campaignItemService.NewCampaignItems(context.TODO(), campaign, *reader); err != nil {
		cs.DeleteCampaign(ctx, campaign)
		return campaign, err
	}

	campaignIds := []primitive.ObjectID{campaign.Id}
	campaign.TotalPendingItems = campaignItemService.GetTotalGroupedPendingItems(ctx, campaignIds)[campaign.Id]

	return campaign, nil
}

func (cs *CampaignService) UpdateCampaign(ctx context.Context, campaign *models.Campaign) error {
	if err := cs.valid(ctx, campaign); err != nil {
		return err
	}

	campaign.UpdatedAt = time.Now()

	hasFilename := campaign.Filename != ""

	if !hasFilename {
		campaignStored, _ := cs.GetCampaign(ctx, campaign.Id.Hex())
		campaign.Filename = campaignStored.Filename
		campaign.CreateAt = campaignStored.CreateAt
	}

	campaign.StartDate = campaign.StartDate.Add(time.Duration(+6) * time.Hour)
	campaign.EndDate = campaign.EndDate.Add(time.Duration(+6) * time.Hour)

	collection := cs.db.Database("gaia").Collection("campaigns")
	filter := bson.M{"_id": campaign.Id}
	update := bson.M{"$set": campaign}

	if _, err := collection.UpdateOne(ctx, filter, update); err != nil {
		return err
	}

	campaignItemService := NewCampaignItemService(&cs.db)
	campaignIds := []primitive.ObjectID{campaign.Id}
	campaign.TotalPendingItems = campaignItemService.GetTotalGroupedPendingItems(ctx, campaignIds)[campaign.Id]

	if !hasFilename {
		return nil
	}

	reader, err := utils.GetReader(campaign.Filename)
	if err != nil {
		return err
	}

	if campaign.Type == models.TemporaryCampaign || (campaign.Type == models.BaseCampaign && !campaign.Upsert) {
		if err := campaignItemService.DeleteCampaignItems(ctx, *campaign); err != nil {
			return err
		}

		if err := campaignItemService.NewCampaignItems(ctx, *campaign, *reader); err != nil {
			return err
		}
	}

	if campaign.Type == models.BaseCampaign && campaign.Upsert {
		if err := campaignItemService.UpdateCampaignItems(ctx, *campaign, *reader); err != nil {
			return err
		}
	}

	campaign.TotalPendingItems = campaignItemService.GetTotalGroupedPendingItems(ctx, campaignIds)[campaign.Id]

	return nil
}

func (cs *CampaignService) DeleteCampaign(ctx context.Context, campaign models.Campaign) error {
	collection := cs.db.Database("gaia").Collection("campaigns")
	if _, err := collection.DeleteOne(ctx, bson.M{"_id": campaign.Id}); err != nil {
		return err
	}

	campaignItemService := NewCampaignItemService(&cs.db)
	if err := campaignItemService.DeleteCampaignItems(ctx, campaign); err != nil {
		return err
	}

	return nil
}

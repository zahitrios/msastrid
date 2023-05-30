package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/models"
	"ms-astrid/products/paginator"
	"ms-astrid/products/queues"
	"ms-astrid/products/utils"
)

const defaultPage = 1

type GaiaGroupsService struct {
	BaseService
	db             mongo.Client
	m              *sync.RWMutex
	wg             *sync.WaitGroup
	simpleProducts map[string]models.Sku
	skus           []interface{}
	someoneChanged bool
}

func NewGaiaGroupsService(db *mongo.Client) *GaiaGroupsService {
	m := &sync.RWMutex{}
	wg := &sync.WaitGroup{}

	return &GaiaGroupsService{
		db: *db,
		m:  m,
		wg: wg,
	}
}

func (s *GaiaGroupsService) CalculatePrices(ctx context.Context) {
	s.groupSimpleProducts(ctx)

	collection := s.db.Database("gaia").Collection("gaia_groups")
	collection.UpdateMany(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"processed": false}},
	)

	cursor, _ := collection.Find(ctx, bson.M{})

	s.skus = nil

	for cursor.Next(ctx) {
		var gaiaGroup models.GaiaGroup
		cursor.Decode(&gaiaGroup)
		s.wg.Add(1)
		go s.calculatePrice(gaiaGroup)
	}

	s.wg.Wait()

	if !s.someoneChanged {
		return
	}

	if _, err := collection.DeleteMany(ctx, bson.M{"processed": false}); err != nil {
		log.Println("error cleaning gaia groups: ", err)
	}

	s.persistGaiaGroups(ctx, s.skus)
}

func (s *GaiaGroupsService) DeleteGaiaGroups(ctx context.Context) error {
	_, err := s.db.Database("gaia").Collection("gaia_groups").DeleteMany(ctx, bson.M{})
	return err
}

func (p *GaiaGroupsService) GetGaiaGroup(ctx context.Context, sku string) (models.GaiaGroup, error) {
	collection := p.db.Database("gaia").Collection("gaia_groups")

	var skuModel models.GaiaGroup
	err := collection.FindOne(ctx, bson.M{"parent_sku": sku}).Decode(&skuModel)

	return skuModel, err
}

func (s *GaiaGroupsService) GetGaiaGroups(ctx context.Context, filter bson.M, r *http.Request) (*paginator.Paginator, error) {
	collection := s.db.Database("gaia").Collection("gaia_groups")
	total, _ := collection.CountDocuments(ctx, filter)
	cursor, err := collection.Find(ctx, filter, s.opts)

	if err != nil {
		return nil, err
	}

	var gaiaGroups []interface{}
	for cursor.Next(ctx) {
		var gaiaGroup models.GaiaGroup
		cursor.Decode(&gaiaGroup)
		gaiaGroups = append(gaiaGroups, gaiaGroup)
	}

	return paginator.NewPaginator(gaiaGroups, total, r), nil
}

func (s *GaiaGroupsService) GetAll(ctx context.Context, filter bson.M) ([]models.GaiaGroup, error) {
	var gaiaGroups []models.GaiaGroup
	collection := s.db.Database("gaia").Collection("gaia_groups")
	cursor, err := collection.Find(ctx, filter, s.opts)

	if err == nil {
		err = cursor.All(ctx, &gaiaGroups)
	}

	return gaiaGroups, err
}

func (s *GaiaGroupsService) PublishGaiaGroups(ctx context.Context) error {
	q, err := queues.NewPublisher()

	if err != nil {
		return err
	}

	filter := bson.M{
		"price": bson.M{
			"$gt": 0,
		},
		"published": false,
	}

	collection := s.db.Database("gaia").Collection("gaia_groups")
	cursor, err := collection.Find(ctx, filter)

	if err != nil {
		return err
	}

	var (
		skusToPublish []interface{}
		skusToUpdate  []string
	)

	for cursor.Next(ctx) {
		var sku models.GaiaGroupSkuRabbit
		cursor.Decode(&sku)
		skusToPublish = append(skusToPublish, sku)
		skusToUpdate = append(skusToUpdate, sku.Sku)
	}

	if len(skusToPublish) > 0 {
		q.PublishSkus(skusToPublish)
	}

	collection.UpdateMany(
		ctx,
		bson.M{"parent_sku": bson.M{"$in": skusToUpdate}},
		bson.M{"$set": bson.M{"published": true}},
	)

	return nil
}

func (s *GaiaGroupsService) SyncGaiaGroups(ctx context.Context) error {
	s.DeleteGaiaGroups(ctx)

	s.wg.Add(1)

	gaiaGroups, err := s.processPage(ctx, defaultPage)
	if err != nil {
		log.Println("error processing page first time: ", err)
		return err
	}

	pages := gaiaGroups.Pagination.Pages
	for page := 2; page <= pages; page++ {
		s.wg.Add(1)
		s.processPage(ctx, page)
	}

	s.wg.Wait()

	return nil
}

func (s *GaiaGroupsService) calculatePrice(gaiaGroup models.GaiaGroup) {
	var (
		msrp  float64
		price float64
		cost  float64
	)

	gaiaGroup.Published = true

	for _, child := range gaiaGroup.Children {
		simpleProduct, ok := s.simpleProducts[child.Sku]
		if !ok {
			continue
		}

		qty, _ := strconv.Atoi(child.Qty)

		listPrice := simpleProduct.Price
		if simpleProduct.Msrp > 0 {
			listPrice = simpleProduct.Msrp
		}

		if !simpleProduct.Published {
			s.someoneChanged = true
			gaiaGroup.Published = false
		}

		msrp += float64(qty) * listPrice
		price += float64(qty) * simpleProduct.Price
		cost += float64(qty) * simpleProduct.Cost
	}

	if price > 0 {
		gaiaGroup.Msrp = msrp
		gaiaGroup.Price = s.roundToNine(price * .95)
		gaiaGroup.Cost = cost
	}

	gaiaGroup.Processed = true

	s.m.Lock()
	s.skus = append(s.skus, gaiaGroup)
	s.m.Unlock()

	s.wg.Done()
}

func (s *GaiaGroupsService) groupSimpleProducts(ctx context.Context) {
	cursor, _ := s.db.Database("gaia").Collection("price_list").Find(ctx, bson.M{})
	s.simpleProducts = map[string]models.Sku{}

	for cursor.Next(ctx) {
		var sku models.Sku
		cursor.Decode(&sku)
		s.simpleProducts[sku.Sku] = sku
	}
}

func (s *GaiaGroupsService) processPage(ctx context.Context, page int) (models.GaiaGroups, error) {
	var gaiaGroupes models.GaiaGroups

	url := fmt.Sprintf("%s%d", os.Getenv("GAIA_GROUPS_ENDPOINT"), page)
	req, err := utils.MakeRequest(url, http.MethodGet, nil)
	if err != nil {
		s.wg.Done()
		return gaiaGroupes, err
	}

	body := req.Body
	jsonAsString, err := io.ReadAll(body)
	if err != nil {
		s.wg.Done()
		return gaiaGroupes, err
	}

	if err := json.Unmarshal(jsonAsString, &gaiaGroupes); err != nil {
		s.wg.Done()
		return gaiaGroupes, err
	}

	var skus []interface{}
	for _, sku := range gaiaGroupes.Skus {
		skus = append(skus, sku)
	}

	if err := s.persistGaiaGroups(ctx, skus); err != nil {
		s.wg.Done()
		return gaiaGroupes, nil
	}

	s.wg.Done()

	return gaiaGroupes, nil
}

func (s *GaiaGroupsService) persistGaiaGroups(ctx context.Context, skus []interface{}) error {
	collection := s.db.Database("gaia").Collection("gaia_groups")
	_, err := collection.InsertMany(ctx, skus)
	return err
}

func (s *GaiaGroupsService) roundToNine(price float64) float64 {
	priceAsString := fmt.Sprintf("%d", int(price))
	priceLength := len(priceAsString)

	lastNumber, _ := strconv.Atoi(priceAsString[priceLength-1 : priceLength])
	diff := float64(9 - lastNumber)

	finalPrice := price + diff
	withoutDecimals := int(finalPrice)

	return float64(withoutDecimals)
}

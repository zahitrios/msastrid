package models

import (
	"errors"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	ApprovedStatus = "approved"
	PenddingStatus = "pendding"
)

const (
	ErrCostUpdated = "cost was updated is needed to review"
)

const (
	iva             = 1.16
	marginToApprove = 30
)

var (
	errCostInvalid   = errors.New("cost is invalid")
	errPriceInvalid  = errors.New("price is invalid")
	errProfitInvalid = errors.New("profit is invalid")
	profit           float64
)

type CampaignItem struct {
	Id        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Campaign  primitive.ObjectID `bson:"campaign" json:"campaign"`
	Priority  int                `bson:"priority" json:"priority"`
	Sku       string             `bson:"sku" json:"sku"`
	Price     float64            `bson:"price" json:"price"`
	Msrp      float64            `bson:"msrp" json:"msrp"`
	Cost      float64            `bson:"cost" json:"cost"`
	NewCost   float64            `bson:"new_cost" json:"new_cost"`
	Profit    float64            `bson:"profit" json:"profit"`
	Status    string             `bson:"status" json:"status"`
	Err       string             `bson:"error" json:"error"`
	CreateAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"update_at" json:"update_at"`
}

type CampaignItemGrouped struct {
	Id            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Priority      int                `bson:"priority" json:"priority"`
	CampaignItems []CampaignItem     `json:"campaign_items" bson:"campaign_items"`
}

type DetailOfPendingItems struct {
	TotalUnder30            int64 `json:"total_under_30"`
	TotalWithNegativeOrZero int64 `json:"total_with_negative_or_zero"`
}

type PendingItems struct {
	Campaign primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Total    int                `bson:"total_pending_items" json:"total_pending_items"`
}

func NewCampaignItem(
	campaign Campaign,
	sku string,
	cost, msrp, price float64) *CampaignItem {

	ci := &CampaignItem{
		Campaign: campaign.Id,
		Priority: campaign.Priority,
		Cost:     cost,
		Msrp:     msrp,
		Price:    price,
		Sku:      sku,
	}

	ci.CreateAt = time.Now()
	ci.UpdatedAt = time.Now()

	if err := ci.valid(); err != nil {
		ci.Err = err.Error()
		ci.Status = PenddingStatus

		return ci
	}

	ci.Status = ApprovedStatus
	ci.Profit = ci.calculateMargin()

	if !ci.isValidMargin() {
		ci.Err = errProfitInvalid.Error()
		ci.Status = PenddingStatus
	}

	return ci
}

func (ci *CampaignItem) valid() error {
	if !ci.isValidCost() {
		return errCostInvalid
	}

	if !ci.isValidPrice() {
		return errPriceInvalid
	}

	return nil
}

func (ci *CampaignItem) calculateMargin() float64 {
	profit = ((1 - (ci.Cost / (ci.Price / iva))) * 100)

	return math.Floor(profit*100) / 100
}

func (ci *CampaignItem) isValidCost() bool {
	return ci.Cost > 0
}

func (ci *CampaignItem) isValidPrice() bool {
	return ci.Price > 0
}

func (ci *CampaignItem) isValidMargin() bool {
	return ci.calculateMargin() > marginToApprove
}

package models

import (
	"ms-astrid/products/horrors"
	"ms-astrid/products/utils"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	BaseCampaign      = "base"
	TemporaryCampaign = "temporal"
)

type Campaign struct {
	Id                primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name              string             `bson:"name" json:"name"`
	StartDate         time.Time          `bson:"start_date" json:"start_date"`
	EndDate           time.Time          `bson:"end_date" json:"end_date"`
	Priority          int                `bson:"priority" json:"priority"`
	Filename          string             `bson:"filename" json:"filename"`
	Type              string             `bson:"type" json:"type"`
	Enable            bool               `bson:"enable" json:"enable"`
	Applied           bool               `bson:"applied" json:"applied"`
	TotalPendingItems int                `json:"total_pending_items"`
	Upsert            bool               `json:"upsert,omitempty"`
	CreateAt          time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

func (c *Campaign) validType() error {
	if !(c.Type == BaseCampaign || c.Type == TemporaryCampaign) {
		return horrors.NewBadRequestError(utils.Translate("campaign_type_is_not_valid"))
	}

	return nil
}

func (c *Campaign) ConvertDates() {
	loc, _ := time.LoadLocation("America/Mexico_City")

	c.StartDate = c.StartDate.In(loc)
	c.EndDate = c.EndDate.In(loc)

	c.CreateAt = c.CreateAt.In(loc)
	c.UpdatedAt = c.UpdatedAt.In(loc)
}

func (c *Campaign) Valid() error {
	if err := c.validType(); err != nil {
		return err
	}

	if c.Name == "" {
		return horrors.NewBadRequestError(utils.Translate("name_is_required"))
	}

	if !c.Id.IsZero() {
		if c.Filename = strings.TrimSpace(c.Filename); c.Filename == "" {
			return nil
		}
	}

	if _, err := url.ParseRequestURI(c.Filename); err != nil {
		return horrors.NewBadRequestError(utils.Translate("filename_is_not_valid"))
	}

	return nil
}

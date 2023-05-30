package models

type Sku struct {
	Sku       string  `bson:"sku" json:"sku"`
	Price     float64 `bson:"price" json:"price"`
	Msrp      float64 `bson:"msrp" json:"msrp"`
	Cost      float64 `bson:"cost" json:"cost"`
	Priority  int     `bson:"priority" json:"priority"`
	Published bool    `bson:"published" json:"published"`
}

type SkuList struct {
	Skus []string `json:"skus"`
}

type SkuListGroup struct {
	Sku       string `bson:"sku" json:"sku"`
	PriceList []Sku  `json:"price_list" bson:"price_list"`
}

type SkuRabbit struct {
	Sku   string  `bson:"sku" json:"sku"`
	Price float64 `bson:"price" json:"price"`
	Msrp  float64 `bson:"msrp" json:"msrp"`
	Cost  float64 `bson:"cost" json:"cost"`
}

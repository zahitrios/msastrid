package models

type GaiaGroupSku struct {
	Qty string `bson:"qty" json:"qty"`
	Sku string `bson:"sku" json:"sku"`
}

type GaiaGroup struct {
	ParentSku string         `bson:"parent_sku" json:"parent_sku"`
	Price     float64        `bson:"price" json:"price"`
	Msrp      float64        `bson:"msrp" json:"msrp"`
	Cost      float64        `bson:"cost" json:"cost"`
	Children  []GaiaGroupSku `bson:"children" json:"children"`
	Published bool           `bson:"published" json:"published"`
	Processed bool           `bson:"processed" json:"processed,omitempty"`
}

type GaiaGroupPagination struct {
	Total int `bson:"total" json:"total"`
	Pages int `bson:"pages" json:"pages"`
	Count int `bson:"count" json:"count"`
	Page  int `bson:"page" json:"page"`
}

type GaiaGroups struct {
	Skus       []GaiaGroup         `bson:"skus" json:"skus"`
	Pagination GaiaGroupPagination `json:"pagination"`
}

type GaiaGroupSkuRabbit struct {
	Sku   string  `bson:"parent_sku" json:"sku"`
	Price float64 `bson:"price" json:"price"`
	Msrp  float64 `json:"msrp"`
	Cost  float64 `bson:"cost" json:"cost"`
}

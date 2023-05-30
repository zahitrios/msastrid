package request

// ItemPriceAstrid struct for response item astrid
type ItemPriceAstrid struct {
	Sku       string  `json:"sku"`
	SkuParent string  `json:"parent_sku"`
	Price     float64 `json:"price"`
	Msrp      float64 `json:"msrp"`
	Cost      float64 `json:"cost"`
}

// ResponseAstridPriceList struct for response list astrid
type ResponseAstridPriceList struct {
	Total   int               `json:"total"`
	Pages   int               `json:"pages"`
	Results int               `json:"results"`
	Data    []ItemPriceAstrid `json:"data"`
}

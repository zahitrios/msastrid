package request

// Price chain struct for price
type Price struct {
	Default float64 `json:"default"`
}
type Generales struct {
	Comercializable bool `json:"comercializable"`
}

// PriceItemCurrency chain struct for price
type PriceItemCurrency struct {
	MXN Price `json:"MXN"`
}
type Enhanced struct {
	Generales Generales `json:"generales"`
}

// ItemBrowseAlgolia struct for item browse request in algolia
type ItemBrowseAlgolia struct {
	Sku                string            `json:"sku"`
	Name               string            `json:"name"`
	Price              PriceItemCurrency `json:"price"`
	Msrp               float64           `json:"msrp"`
	IsEnabled          bool              `json:"is_enabled"`
	IsAvailable        bool              `json:"is_available"`
	EnhancedAttributes Enhanced          `json:"enhanced_attributes"`
}

// ResponseBrowseAlgolia see https://www.algolia.com/doc/rest-api/search/#browse-index-get
// ResponseBrowseAlgolia struct for browse request in algolia
type ResponseBrowseAlgolia struct {
	Pages  int                 `json:"nbPages"`
	Hits   []ItemBrowseAlgolia `json:"hits"`
	Cursor string              `json:"cursor"`
	Page   int                 `json:"page"`
}

// AlgoliaParameters queryParams for request in algolia
type AlgoliaParameters struct {
	Index         string
	ApplicationId string
	ApiKey        string
}

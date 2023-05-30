package models

// ItemMergePrice struct for response merge price report
type ItemMergePrice struct {
	Sku             string  `json:"sku"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	Msrp            float64 `json:"msrp"`
	M2              float64 `json:"m2"`
	M2Change        int     `json:"m2-change"`
	Gama            string  `json:"gama"`
	GamaChange      int     `json:"gama-change"`
	Cost            float64 `json:"cost"`
	IsEnabled       bool    `json:"is_enabled"`
	IsAvailable     bool    `json:"is_available"`
	Comercializable bool    `json:"comercializable"`
}

package models

type SeGroupInventory struct {
	Count   int64         `json:"count"`
	Results []interface{} `json:"results"`
}

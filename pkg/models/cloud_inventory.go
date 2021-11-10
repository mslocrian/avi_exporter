package models

type CloudInventory struct {
	Count   int64         `json:"count"`
	Results []interface{} `json:"results"`
}

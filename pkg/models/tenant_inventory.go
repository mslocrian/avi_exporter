package models

type TenantInventory struct {
	Count   int64         `json:"count"`
	Results []interface{} `json:"results"`
}

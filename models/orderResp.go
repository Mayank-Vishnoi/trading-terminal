package models

type OrderResp struct {
	Status string `json:"status"`
	Data   struct {
		OrderID string `json:"order_id"`
	} `json:"data"`
}

package models

type OrderRequest struct {
	Quantity          int     `json:"quantity"`
	Product           string  `json:"product"`
	Validity          string  `json:"validity"`
	Price             float64 `json:"price"`
	Tag               string  `json:"tag"`
	InstrumentToken    string  `json:"instrument_token"`
	OrderType         string  `json:"order_type"`
	TransactionType   string  `json:"transaction_type"`
	DisclosedQuantity int     `json:"disclosed_quantity"`
	TriggerPrice      float64 `json:"trigger_price"`
	IsAmo             bool    `json:"is_amo"`
}

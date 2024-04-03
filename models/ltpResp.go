package models

type InstrumentData struct {
	LastPrice       float64 `json:"last_price"`
	InstrumentToken string  `json:"instrument_token"`
}

type LtpResponse struct {
	Status string                    `json:"status"`
	Data   map[string]InstrumentData `json:"data"`
}
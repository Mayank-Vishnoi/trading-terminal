package models

type OptionChainResp struct {
	Status string       `json:"status"`
	Data   []OptionData `json:"data"` // Use slice for array of options
}

type OptionData struct {
	Expiry              string         `json:"expiry"`
	Pcr                 float64        `json:"pcr"`
	StrikePrice         float64        `json:"strike_price"`
	UnderlyingKey       string         `json:"underlying_key"`
	UnderlyingSpotPrice float64        `json:"underlying_spot_price"`
	CallOptions         CallOptionData `json:"call_options"`
	PutOptions          PutOptionData  `json:"put_options"`
}

type CallOptionData struct {
	InstrumentKey string       `json:"instrument_key"`
	MarketData    MarketData   `json:"market_data"`
	OptionGreeks  OptionGreeks `json:"option_greeks"`
}

type PutOptionData struct {
	InstrumentKey string       `json:"instrument_key"`
	MarketData    MarketData   `json:"market_data"`
	OptionGreeks  OptionGreeks `json:"option_greeks"`
}

type MarketData struct {
	Ltp        float64 `json:"ltp"`
	ClosePrice float64 `json:"close_price"`
	Volume     float64 `json:"volume"`
	Oi         float64 `json:"oi"`
	BidPrice   float64 `json:"bid_price"`
	BidQty     float64 `json:"bid_qty"`
	AskPrice   float64 `json:"ask_price"`
	AskQty     float64 `json:"ask_qty"`
	PrevOi     float64 `json:"prev_oi"`
}

type OptionGreeks struct {
	Vega  float64 `json:"vega"`
	Theta float64 `json:"theta"`
	Gamma float64 `json:"gamma"`
	Delta float64 `json:"delta"`
	Iv    float64 `json:"iv"`
}

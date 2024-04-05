package orders

import "net/http"

// Places an order with triggers on underlying price and would shift stoploss as the target is approaching
func PlaceTrailingStopLossOrder(w http.ResponseWriter, r *http.Request) {
	
}
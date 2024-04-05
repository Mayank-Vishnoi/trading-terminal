package orders

import "net/http"

// Places an order with triggers on underlying price at lower quantity and would add more quantity based on reaffirmations on target/stoploss
func PartialOrder(w http.ResponseWriter, r *http.Request) {
	
}
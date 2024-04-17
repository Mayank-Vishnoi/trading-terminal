package orders

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// alternate approach: for better modularity
// placeUT(entry, target, sl, ..) usign helper
// if placed: placeUT(reaffirmation, newTarget, newStoploss, ..) and placeUT(entry - 0.9*(entry - stoploss), target, stoploss, ..)
// if placed in one of the above, exit both routiunes
// complicated and having a lot of extra calls to their api

// Places an order with triggers on underlying price at lower quantity and would add more quantity based on reaffirmations or stoploss
func PlacePartialOrder(w http.ResponseWriter, r *http.Request) {
	entry, target, stoploss := r.URL.Query().Get("entry"), r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	targetPrice, _ := strconv.ParseFloat(target, 64)
	stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
	duration, _ := strconv.Atoi(r.URL.Query().Get("duration"))
	reaffirmation, _ := strconv.ParseFloat(r.URL.Query().Get("reaffirmation"), 64)
	newStoploss, _ := strconv.ParseFloat(r.URL.Query().Get("newStoploss"), 64)
	newTarget, _ := strconv.ParseFloat(r.URL.Query().Get("newTarget"), 64)
	// newTarget := targetPrice + (reaffirmation - newStoploss) // keep target at least 1:1

	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key)
	use(quantity)
	// use quantity * 3 for normal ut orders, here we use quantity and quantity*2 after reaffirmation
	if underlying_instrument_key == "" {
		return
	}

	var ltp float64
	ltp = getLtp(underlying_instrument_key)
	if ltp == 0 {
		return
	}
	var checkEntryCondition func() bool
	if entryPrice < ltp {
		checkEntryCondition = func() bool {
			return ltp <= entryPrice
		}
	} else {
		checkEntryCondition = func() bool {
			return ltp > entryPrice
		}
	}

	ticker := time.NewTicker(2 * time.Second)
	timeout := time.NewTimer(15 * time.Minute)
	done := make(chan bool)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-timeout.C:
				fmt.Println("Timeout..")
				done <- true
			case <-ticker.C:
				ltp = getLtp(underlying_instrument_key)
				fmt.Println("LTP: ", ltp)
				if checkEntryCondition() {
					// Commented just for testing purposes
					// placeOrder(quantity, instrument_key, "BUY")
					fmt.Println("Order placed on paper")
					done <- true
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	<-done
	quit <- true

	lotsAdded := false
	checkExitCondition := func() bool {
		if entryPrice > stoplossPrice {
			return ltp >= targetPrice || ltp <= stoplossPrice
		} else {
			return ltp <= targetPrice || ltp >= stoplossPrice
		}
	}

	checkAddAtSL := func() bool {
		// if we have hit around 90% closer to stoploss from entry, return true
		if entryPrice < stoplossPrice {
			return ltp >= entryPrice + 0.9*(stoplossPrice-entryPrice)
		} else {
			return ltp <= entryPrice - 0.9*(entryPrice-stoplossPrice)
		}
	}

	checkAddAtReaffirmation := func() bool {
		if entryPrice < stoplossPrice {
			return ltp <= reaffirmation
		} else {
			return ltp >= reaffirmation
		}
	}

	ticker = time.NewTicker(2 * time.Second)
	timeout = time.NewTimer(time.Duration(duration) * time.Minute)
	done = make(chan bool)
	quit = make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				ltp = getLtp(underlying_instrument_key)
				fmt.Println("LTP: ", ltp)
				if !lotsAdded && checkAddAtSL() {
					// placeOrder(quantity*2, instrument_key, "BUY")
					lotsAdded = true
					fmt.Println("Added more lots at stoploss")
				}
				if !lotsAdded && checkAddAtReaffirmation() {
					// placeOrder(quantity*2, instrument_key, "BUY")
					stoplossPrice = newStoploss
					targetPrice = newTarget
					lotsAdded = true
					fmt.Println("Added more lots at reaffirmation")
				}
				if checkExitCondition() {
					// this will never happen before adding lots
					// placeOrder(quantity*3, instrument_key, "SELL")
					fmt.Println("Position squared off due to exit conditions")
					done <- true
				}
			case <-timeout.C:
				if lotsAdded {
					// placeOrder(quantity*3, instrument_key, "SELL")
					use(lotsAdded)
				} else {
					// placeOrder(quantity, instrument_key, "SELL")
					use(lotsAdded)
				}	
				fmt.Println("Exiting position due to timeout")
				done <- true
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	<-done
	quit <- true
}
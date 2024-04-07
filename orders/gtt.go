package orders

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Places an order with triggers on underlying price
func PlaceUTOrder(w http.ResponseWriter, r *http.Request) {
	// read query params
	entry, target, stoploss := r.URL.Query().Get("entry"), r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	targetPrice, _ := strconv.ParseFloat(target, 64)
	stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
	duration, _ := strconv.Atoi(r.URL.Query().Get("duration"))
	fmt.Println("Entry: ", entryPrice, ", Target: ", targetPrice, ", Stoploss: ", stoplossPrice, ", Duration: ", duration)

	// get underlying instrument details
	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key)
	if underlying_instrument_key == "" {
		return
	}

	// get access token
	access_token := getAccessToken()
	if access_token == "" {
		return
	}
	
	// define entry condition
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

	// define goroutine to check entry condition on each ticker until timeout
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
				fmt.Println("LTP: ", ltp, " seeking entry...")
				if checkEntryCondition() {
					fmt.Println("Entry condition met")
					placeOrder(quantity, instrument_key, "BUY")
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


	// define goroutine to exit current position based off target and stoploss
	checkExitCondition := func() bool {
		if targetPrice > stoplossPrice {
			return ltp >= targetPrice || ltp <= stoplossPrice
		} else {
			return ltp <= targetPrice || ltp >= stoplossPrice
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
				// fetch(): updaets target, stoploss
				ltp = getLtp(underlying_instrument_key)
				fmt.Println("LTP: ", ltp, " seeking exit...")
				if checkExitCondition() {
					fmt.Println("Exit condition met")
					placeOrder(quantity, instrument_key, "SELL")
					done <- true
				}
			case <-timeout.C:
				fmt.Println("Timeout.. exiting from order regardless of target or stoploss")
				placeOrder(quantity, instrument_key, "SELL")
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
package orders

import (
	"context"
	"fmt"
	"myapp/config"
	"net/http"
	"os"
	"strconv"
	"time"
)

/*
1) Redis
Create another internal API to update target/stoploss for an instrument in redis.
Use that to update the target/stoploss inside the routine as one more step of fetching redis along with the ltp.
*/

/*
2) Different orders: but maintain order_id in redis anyway and use external api to cancel/modify but better semantics this way
buy order - entry
sell order - exit target => exit stoploss order, enter new stoploss order
sell order - exit stoploss
*/

// Places an order with triggers on underlying price but being able to modify target and stoploss in case of profit booking
func PlaceTrailingOrder(w http.ResponseWriter, r *http.Request) {
	// read query params
	entry, target, stoploss := r.URL.Query().Get("entry"), r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	duration, _ := strconv.Atoi(r.URL.Query().Get("duration"))
	fmt.Println("Entry: ", entryPrice, ", Target: ", target, ", Stoploss: ", stoploss, ", Duration: ", duration)

	// get underlying instrument details
	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key)
	use(quantity) // commented for testing, clean and commit after successful testing
	if underlying_instrument_key == "" {
		return
	}

	redisClient, err := config.NewRedisClient(context.Background(), os.Getenv("REDIS_URL"))
	if err != nil {
		fmt.Println("Error initializing Redis client:", err)
		return
	}
	_ = redisClient.SetVal(instrument_key + "_target", target)
	_ = redisClient.SetVal(instrument_key + "_stoploss", stoploss)
	_ = redisClient.SetVal(instrument_key + "_entry", entry)
	
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
				fmt.Println("LTP: ", ltp)
				if checkEntryCondition() {
					fmt.Println("Entry condition met, placing buy order...")
					// placeOrder(quantity, instrument_key, "BUY")
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

	// get perhaps updated target and stoploss: 
	// but is there a way to get them by another signal in the routine instead of fetching them every 2 seconds?
	checkExitCondition := func() bool {
		start_time := time.Now().UnixNano()/int64(time.Millisecond)
		target, _ = redisClient.GetVal(instrument_key + "_target")
		end_time := time.Now().UnixNano()/int64(time.Millisecond)
		fmt.Println("Time taken to fetch target from redis(without the initialization of client): ", end_time - start_time)
		stoploss, _ = redisClient.GetVal(instrument_key + "_stoploss")
		targetPrice, _ := strconv.ParseFloat(target, 64)
		stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
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
				ltp = getLtp(underlying_instrument_key)
				fmt.Println("LTP: ", ltp)
				if checkExitCondition() {
					fmt.Println("Exit condition met, placing sell order...")
					// placeOrder(quantity, instrument_key, "SELL")
					done <- true
				}
			case <-timeout.C:
				fmt.Println("Timeout! Exiting from order regardless of target or stoploss...")
				// placeOrder(quantity, instrument_key, "SELL")
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

func ModifyTrailingOrder(w http.ResponseWriter, r *http.Request) {
	target, stoploss := r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	targetPrice, _ := strconv.ParseFloat(target, 64)
	stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
	instrument_key := r.URL.Query().Get("instrument_key")
	
	// init redis
	redisClient, err := config.NewRedisClient(context.Background(), os.Getenv("REDIS_URL"))
	if err != nil {
		fmt.Println("Error initializing Redis client:", err)
		return
	}
	
	// sanity checks
	// invalid parameters: ignore for now
	// making sure newTarget is bigger than old
	oldTarget, _ := redisClient.GetVal(instrument_key + "_target")
	oldStoploss, err := redisClient.GetVal(instrument_key + "_stoploss")
	if err != nil {
		fmt.Println("Error getting old target and/or stoploss:", err)
		return
	}
	oldTargetPrice, _ := strconv.ParseFloat(oldTarget, 64)
	oldStoplossPrice, _ := strconv.ParseFloat(oldStoploss, 64)
	if oldTargetPrice < oldStoplossPrice {
		if targetPrice >= oldTargetPrice {
			fmt.Println("New target should be bigger than old target")
			return
		}
	} else {
		if targetPrice <= oldTargetPrice {
			fmt.Println("New target should be bigger than old target")
			return
		}
	}

	// making sure that we book at least 1:1 profit before shifting stoploss
	entry, _ := redisClient.GetVal(instrument_key + "_entry")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	if stoplossPrice - entryPrice < entryPrice - oldStoplossPrice {
		fmt.Println("Cannot shift stoploss before booking 1:1 profit from entry price")
		return
	}

	// make sure this gets placed only when we already have enough target from entry
	underlying_instrument_key, _ := getUnderlyingDetails(instrument_key)
	ltp := getLtp(underlying_instrument_key)
	if oldTarget < oldStoploss {
		if ltp - entryPrice < entryPrice - oldStoplossPrice {
			fmt.Println("Cannot shift stoploss before booking 1:1 profit from entry price")
			return
		}
	} else {
		if entryPrice - ltp < oldStoplossPrice - entryPrice {
			fmt.Println("Cannot shift stoploss before booking 1:1 profit from entry price")
			return
		}
	}

	// set target and stoploss
	_ = redisClient.SetVal(instrument_key + "_target", target)
	err = redisClient.SetVal(instrument_key + "_stoploss", stoploss)
	if err != nil {
		fmt.Println("Error setting stoploss and/or target:", err)
		return
	}
}
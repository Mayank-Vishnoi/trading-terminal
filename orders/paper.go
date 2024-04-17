package orders

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// places order in the same way as underlying trigger order, but calculates the potential change in premium as instead of placing the order
func PlaceUTPaperOrder(w http.ResponseWriter, r *http.Request) {
	// use the input from terminal as getLTP(), keep that helper in this function itself?

	entry, target, stoploss := r.URL.Query().Get("entry"), r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	targetPrice, _ := strconv.ParseFloat(target, 64)
	stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
	duration, _ := strconv.Atoi(r.URL.Query().Get("duration"))

	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key)
	if underlying_instrument_key == "" || quantity == 0 {
		return
	}

	// time to inititate connection and fetch from redis is 500ms
	// start_time := time.Now().UnixNano()/int64(time.Millisecond)
	// access_token := getAccessToken()
	// if access_token == "" {
	// 	return
	// }
	// end_time := time.Now().UnixNano()/int64(time.Millisecond)
	// fmt.Println("Time taken to fetch token from redis: ", end_time - start_time)

	var entry_premium, exit_premium, profit, entry_underlying, exit_unlerlying, capital float64
	var entry_time, exit_time int64

	// should I get token outside and pass it as parameter?
	var ltp float64
	// start_time := time.Now().UnixNano()/int64(time.Millisecond)
	ltp = getLtp(underlying_instrument_key)
	// end_time := time.Now().UnixNano()/int64(time.Millisecond)
	// fmt.Println("Time taken to fetch ltp from upstox api(includes the time for getting access token): ", end_time - start_time)
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
					// note down the premium at entry
					entry_premium = getLtp(instrument_key)
					// note down the premium at entry for underlying
					entry_underlying = ltp
					// note down current time
					entry_time = time.Now().Unix()
					// capital = premium * quantity
					capital = entry_premium * float64(quantity)

					fmt.Println("Order placed on paper, at: ", time.Now().Format("03:04:05 PM"))
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

	fmt.Println("sometimes this doesn't get executed after order placed. why is that!")

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
				ltp = getLtp(underlying_instrument_key)
				fmt.Println("LTP: ", ltp)
				if checkExitCondition() {
					// note down the premium at exit
					exit_premium = getLtp(instrument_key)
					// note down the premium at exit for underlying
					exit_unlerlying = ltp
					// profit = (exit_premium - entry_premium) / entry_premium * 100
					profit = (exit_premium - entry_premium) / entry_premium * 100
					// note down current time
					exit_time = time.Now().Unix()
					fmt.Println("Order exited on paper due to exit conditions, at: ", time.Now().Format("03:04:05 PM"))
					done <- true
				}
			case <-timeout.C:
				exit_premium = getLtp(instrument_key)
				profit = (exit_premium - entry_premium) / entry_premium * 100
				exit_time = time.Now().Unix()
				fmt.Println("Order exited on paper due to timeout, at: ", time.Now().Format("03:04:05 PM"))
				done <- true
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	<-done
	quit <- true

	// print all
	fmt.Println("Entry Time: ", entry_time)
	fmt.Println("Entry Underlying: ", entry_underlying)
	fmt.Println("Entry Premium: ", entry_premium)
	fmt.Println("Exit Time: ", exit_time)
	fmt.Println("Exit Premium: ", exit_premium)
	fmt.Println("Exit Underlying: ", exit_unlerlying)
	fmt.Println("Change in underlying: ", (exit_unlerlying - entry_underlying))
	fmt.Println("Change in premium: ", (exit_premium - entry_premium))
	fmt.Println("Capital: ", capital)
	fmt.Println("Profit: ", profit)


	// function to export this data to a csv file? or online?
}
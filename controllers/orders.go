package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"myapp/config"
	"myapp/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// helper function to get underlying details from the instrument key
func getUnderlyingDetails(instrument_key string) (string, int) {
	var underlying_instrument_key string
	var quantity int

	baseURL := "https://api.upstox.com/v2/market-quote/ltp"
	url := fmt.Sprintf("%s?instrument_key=%s", baseURL, instrument_key)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	access_token := getAccessToken()
	req.Header.Add("Authorization", "Bearer "+access_token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error performing the http request: ", err)
		return "", 0
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	// fmt.Println(string(body))

	var ltpResp models.LtpResponse
	err = json.Unmarshal(body, &ltpResp)
	if err != nil {
		fmt.Println("Error unmarshalling response: ", err)
		return "", 0
	}

	for key := range ltpResp.Data {
		if strings.Contains(key, "BANKNIFTY") {
			quantity = 15
			underlying_instrument_key = "NSE_INDEX|Nifty Bank"
		} else if strings.Contains(key, "NIFTY") {
			quantity = 50
			underlying_instrument_key = "NSE_INDEX|Nifty 50"
		} else {
			fmt.Println("Given instrument key is not an index derivative")
		}
	}

	if (underlying_instrument_key == "") {
		fmt.Println(string(body))
	}

	return underlying_instrument_key, quantity
}

// helper function to get access token from redis
func getAccessToken() string {
	redisClient, err := config.NewRedisClient(context.Background(), os.Getenv("REDIS_URL"))
	if err != nil {
		fmt.Println("Error initializing Redis client:", err)
		return ""
	}
	access_token, err := redisClient.GetVal("access_token")
	if err != nil {
		fmt.Println("Error getting access token:", err)
		return ""
	}
	return access_token
}

// helper function to place order
func placeOrder(quantity int, instrument_key string, transaction_type string) {
	order := models.OrderRequest{
		Quantity: quantity,
		Product: "I",
		Validity: "DAY",
		Price: 0,
		Tag: "string",
		InstrumentToken: instrument_key,
		OrderType: "MARKET",
		TransactionType: transaction_type,
		DisclosedQuantity: quantity,
		TriggerPrice: 0,
		IsAmo: false,
	}
	orderJSON, err := json.Marshal(order)
	if err != nil {
		fmt.Println("Error marshalling order request: ", err)
		return
	}

	url := "https://api.upstox.com/v2/order/place"
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(orderJSON))
	access_token := getAccessToken()
	req.Header.Add("Authorization", "Bearer "+access_token)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if (err != nil) {
		fmt.Println("Error performing http request: ", err)
		return
	}
	defer res.Body.Close()
	
	body, _ := io.ReadAll(res.Body)
	var orderResp models.OrderResp
	err = json.Unmarshal(body, &orderResp)
	if err != nil {
		fmt.Println("Error unmarshalling response: ", err)
		return
	}

	if orderResp.Status == "success" {
		fmt.Println("Order placed successfully")
	} else {
		fmt.Println(string(body))
	}
}

// helper function to get last traded price
func getLtp(instrument_key string) float64 {
	baseURL := "https://api.upstox.com/v2/market-quote/ltp"
	url := fmt.Sprintf("%s?instrument_key=%s", baseURL, instrument_key)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	access_token := getAccessToken()
	req.Header.Add("Authorization", "Bearer "+access_token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error performing the http request: ", err)
		return 0
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	var ltpResp models.LtpResponse
	err = json.Unmarshal(body, &ltpResp)
	if err != nil {
		fmt.Println("Error unmarshalling response: ", err)
		return 0
	}

	var ltp float64
	for _, value := range ltpResp.Data {
		ltp = value.LastPrice
		break
	}
	if ltp == 0 {
		fmt.Println(string(body))
	}
	return ltp
}


// Places an order with triggers on underlying price
func PlaceUTOrder(w http.ResponseWriter, r *http.Request) {
	// read query params
	entry, target, stoploss := r.URL.Query().Get("entry"), r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	targetPrice, _ := strconv.ParseFloat(target, 64)
	stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
	fmt.Print("Entry: ", entryPrice, "Target: ", targetPrice, "Stoploss: ", stoplossPrice, "\n")

	// get underlying instrument details
	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key)
	if underlying_instrument_key == "" || quantity == 0 {
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
	timeout = time.NewTimer(30 * time.Minute)
	done = make(chan bool)
	quit = make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
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


// Places an order with triggers on underlying price and would shift stoploss as the target is approaching
func PlaceTrailingStopLossOrder(w http.ResponseWriter, r *http.Request) {

}


// Places an order with triggers on underlying price at lower quantity and would add more quantity based on reaffirmations on target/stoploss
func PartialOrder(w http.ResponseWriter, r *http.Request) {
	
}
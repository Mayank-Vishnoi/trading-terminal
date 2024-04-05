package orders

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
	"strings"
)

// GET /quote/optionchain
func GetOptionChain(instrument_key, expiry_date string) {
	baseURL := "https://api.upstox.com/v2/option/chain"
	url := fmt.Sprintf("%s?instrument_key=%s&expiry_date=%s", baseURL, instrument_key, expiry_date)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	redisClient, err := config.NewRedisClient(context.Background(), os.Getenv("REDIS_URL"))
	if err != nil {
		fmt.Println("Error initializing Redis client:", err)
		return
	}
	access_token, err := redisClient.GetVal("access_token")
	if err != nil {
		fmt.Println("Error getting access token:", err)
		return
	}

	req.Header.Add("Authorization", "Bearer "+access_token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	// fmt.Println(string(body))

	var ltpResp models.OptionChainResp
	err = json.Unmarshal(body, &ltpResp)
	if err != nil {
		fmt.Println("Error parsing:", err)
		return
	}

	for _, data := range ltpResp.Data {
		fmt.Println(data.StrikePrice, "CE: ", data.CallOptions.MarketData.Ltp, "PE: ", data.PutOptions.MarketData.Ltp)
	}
}

// get underlying details from the option instrument key
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

// get access token from redis
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

// place order
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

// get last traded price
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
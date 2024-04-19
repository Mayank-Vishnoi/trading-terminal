package utility

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"myapp/models"
	"net/http"
	"os"
)

// getLTP
func GetLtp(instrument_key string) float64 {
	baseURL := "https://api.upstox.com/v2/market-quote/ltp"
	url := fmt.Sprintf("%s?instrument_key=%s", baseURL, instrument_key)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	access_token := os.Getenv("ACCESS_TOKEN")
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

// helper function to place order
func PlaceOrder(quantity int, instrument_key string, transaction_type string) {
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
	access_token := os.Getenv("ACCESS_TOKEN")
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

func Use(thing any) {
	_ = thing
}
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"myapp/config"
	"myapp/models"
	"net/http"
	"os"
	"time"
)

// GET /quote/optionchain
func GetOptionChain(w http.ResponseWriter, r *http.Request) {
	instrument_key := r.URL.Query().Get("instrument-key")
	expiry_date := r.URL.Query().Get("expiry-date")

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


// GET /quote/ltp - ticker for every 3 seconds | stops after 16 seconds or if price > 47000
func GetLTP(w http.ResponseWriter, r *http.Request) {
	instrument := r.URL.Query().Get("instrument")

	var instrument_key, instrument_name string
	if instrument == "nifty50" {
		instrument_key = "NSE_INDEX|Nifty 50"
		instrument_name = "NSE_INDEX:Nifty 50"
	} else if instrument == "banknifty" {
		instrument_key = "NSE_INDEX|Nifty Bank"
		instrument_name = "NSE_INDEX:Nifty Bank"
	}

	baseURL := "https://api.upstox.com/v2/market-quote/ltp"
	url := fmt.Sprintf("%s?instrument_key=%s", baseURL, instrument_key)

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

	ticker := time.NewTicker(3 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
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
				
				var ltpResp models.LtpResponse
				err = json.Unmarshal(body, &ltpResp)
				if err != nil {
					fmt.Println("Error parsing:", err)
					return
				}

				price := ltpResp.Data[instrument_name].LastPrice
				fmt.Println(price)

				if price > 47000 {
					// do the needful here before stoping ticker
					done <- true
				}

				// fmt.Println("this won't be executed")
			}
		}
	}()

	time.Sleep(16 * time.Second) // fetch 16/3 = 5 times
	ticker.Stop()
	done <- true
}


package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"myapp/models"
	"myapp/workers"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// helper function to get underlying details from the instrument key
func getUnderlyingDetails(instrument_key string) (string, int) {
	var underlying_instrument_key string
	var quantity int

	baseURL := "https://api.upstox.com/v2/market-quote/ltp"
	url := fmt.Sprintf("%s?instrument_key=%s", baseURL, instrument_key)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)

	access_token := os.Getenv("ACCESS_TOKEN")
	if access_token == "" {
		fmt.Println("Access token not found in context")
		return "", 0
	}
	req.Header.Add("Authorization", "Bearer "+access_token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error performing the http request: ", err)
		return "", 0
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var ltpResp models.LtpResponse
	err = json.Unmarshal(body, &ltpResp)
	if err != nil {
		fmt.Println("Error unmarshalling response: ", err)
		return "", 0
	}

	for key := range ltpResp.Data {
		if strings.Contains(key, "BANKNIFTY") {
			quantity, _ = strconv.Atoi(os.Getenv("BANKNIFTY_QUANTITY"))
			underlying_instrument_key = "NSE_INDEX|Nifty Bank"
		} else if strings.Contains(key, "NIFTY") {
			quantity, _ = strconv.Atoi(os.Getenv("NIFTY_QUANTITY"))
			underlying_instrument_key = "NSE_INDEX|Nifty 50"
		} else {
			fmt.Println("Given instrument key is not an index derivative")
		}
	}

	if underlying_instrument_key == "" {
		fmt.Println(string(body))
	}

	return underlying_instrument_key, quantity
}

// Places an order with triggers on underlying price
func PlaceUTOrder(w http.ResponseWriter, r *http.Request) {
	// ctx = r.Context() // this can hold some channels which managerWorker could be passed

	// read query params
	entry, target, stoploss := r.URL.Query().Get("entry"), r.URL.Query().Get("target"), r.URL.Query().Get("stoploss")
	entryPrice, _ := strconv.ParseFloat(entry, 64)
	targetPrice, _ := strconv.ParseFloat(target, 64)
	stoplossPrice, _ := strconv.ParseFloat(stoploss, 64)
	fmt.Print("Entry: ", entryPrice, ", Target: ", targetPrice, ", Stoploss: ", stoplossPrice, "\n")

	if (target > entry && stoploss > entry) || (target < entry && stoploss < entry) {
		fmt.Println("Invalid target/stoploss values")
		return
	}

	// get underlying instrument details
	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key)
	if underlying_instrument_key == "" || quantity == 0 {
		fmt.Println("Error fetching underlying details")
		return
	}

	go func() {
		entryFound := make(chan struct{})
		cancelled := make(chan struct{})

		// create worker
		worker := workers.NewEntryWorker(uuid.New().String(), instrument_key, underlying_instrument_key, quantity, entryPrice, entryFound, cancelled)

		// start worker
		workers.AddWorker(worker)

		select {
		case <-entryFound:
			fmt.Println("Entry found..")
			exitFound := make(chan struct{})
			cancelled := make(chan struct{})
			exitWorker := workers.NewExitWorker(uuid.New().String(), instrument_key, underlying_instrument_key, quantity, targetPrice, stoplossPrice, exitFound, cancelled)
			workers.AddWorker(exitWorker)
			select {
			case <-exitFound:
				fmt.Println("Exit found..")
			case <-cancelled:
				fmt.Println("Exit worker was timed out or cancelled..")
			}
		case <-cancelled:
			fmt.Println("Entry Order was timed out or cancelled..")
		}
	}()

	// used a goroutine above to send response without waiting for workers to finish
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Processing your request.."))
}

// for changes in the exitWorker: target/stoploss (perhaps to trail?)
// for changes in the entryWorker: just cancel a place a new one?
func ModifyExitOrder(w http.ResponseWriter, r *http.Request) {
	// read orderID, target, stoploss from params:
	id := r.URL.Query().Get("id")
	target, _ := strconv.ParseFloat(r.URL.Query().Get("target"), 64)
	stoploss, _ := strconv.ParseFloat(r.URL.Query().Get("stoploss"), 64)
	instrument, underlying, quantity := workers.GetWorkerDetails(id)

	// have to make sure that some target is already obtained before raising it!
	// and new stoploss must be greater than last one
	// TODO

	go func() {
		// cancel associated "exit" worker
		// this seems to work waiting until the worker is cancelled to avoid conditions where exitFound channel receives unwanted signals
		var wg sync.WaitGroup
		if workers.IsExitWorker(id) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				workers.CancelWorker(id)
			}()
		} else {
			fmt.Println("No exit worker found for the given id..")
			return
		}

		// create new exit worker
		wg.Wait()
		exitFound := make(chan struct{})
		cancelled := make(chan struct{})
		worker := workers.NewExitWorker(uuid.New().String(), instrument, underlying, quantity, target, stoploss, exitFound, cancelled)
		workers.AddWorker(worker)

		select {
		case <-exitFound:
			fmt.Println("Exit11 found..")
		case <-cancelled:
			fmt.Println("12Exit worker was timed out or cancelled..")
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Processing your request.."))
}

// TODO
func PlacePartialOrder(w http.ResponseWriter, r *http.Request) {
	// read query params

	// fetch underlying details, quantity

	// create new worker with quantity/3

	// wait for entry

	// upon entry create new worker with 2/3 quantity - at 0.9*entry and reaffirm points

	// wait for corresponding exits after any of one is met

}

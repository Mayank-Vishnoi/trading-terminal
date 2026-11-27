package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"myapp/models"
	"myapp/utility"
	"myapp/workers"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// helper function to get underlying details from the instrument key
func getUnderlyingDetails(instrument_key string, partial bool) (string, int) {
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
			underlying_instrument_key = "NSE_INDEX|Nifty Bank"
			quantity, _ = strconv.Atoi(os.Getenv("BANKNIFTY_LOTS"))
			if partial {
				quantity = quantity / 3
			}
			quantity = quantity * 15
		} else if strings.Contains(key, "NIFTY") {
			underlying_instrument_key = "NSE_INDEX|Nifty 50"
			quantity, _ = strconv.Atoi(os.Getenv("NIFTY_LOTS"))
			if partial {
				quantity = quantity / 3
			}
			quantity = quantity * 50
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
	entry, _ := strconv.ParseFloat(r.URL.Query().Get("entry"), 64)
	target, _ := strconv.ParseFloat(r.URL.Query().Get("target"), 64)
	stoploss, _ := strconv.ParseFloat(r.URL.Query().Get("stoploss"), 64)
	fmt.Print("Entry: ", entry, ", Target: ", target, ", Stoploss: ", stoploss, "\n")

	// get underlying instrument details
	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key, false)
	if underlying_instrument_key == "" || quantity == 0 {
		fmt.Println("Error fetching underlying details")
		return
	}

	// Validations
	if (target > entry && stoploss > entry) || (target < entry && stoploss < entry) {
		fmt.Println("Invalid target/stoploss values")
		return
	}
	if target > entry {
		if target-entry < entry-stoploss {
			fmt.Println("Target/stoploss should at least be 1:1")
			return
		}
	} else {
		if entry-target < stoploss-entry {
			fmt.Println("Target/stoploss should at least be 1:1")
			return
		}
	}

	go func() {
		entryFound := make(chan struct{})
		cancelled := make(chan struct{})
		defer close(entryFound)
		defer close(cancelled)

		// create worker
		worker := workers.NewEntryWorker(uuid.New().String(), instrument_key, underlying_instrument_key, quantity, entry, entryFound, cancelled)

		// start worker
		workers.AddWorker(worker)

		select {
		case <-entryFound:
			fmt.Println("Entry found for UT order!..")
			go startExitWorker(instrument_key, underlying_instrument_key, quantity, target, stoploss)
		case <-cancelled:
			fmt.Println("Entry Order was timed out or cancelled..")
		}
	}()

	// used a goroutine above to send response without waiting for workers to finish
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Processing your request.."))
}

// to modify the exit order: target, stoploss
func ModifyExitOrder(w http.ResponseWriter, r *http.Request) {
	// If you wish to modify the entry order, you can simply cancel the existing entry order and place a new one

	// read orderID, target, stoploss from params:
	id := r.URL.Query().Get("id")
	target, _ := strconv.ParseFloat(r.URL.Query().Get("target"), 64)
	stoploss, _ := strconv.ParseFloat(r.URL.Query().Get("stoploss"), 64)
	instrument, underlying, quantity, oldTarget, oldStoploss := workers.GetWorkerDetails(id)

	// Validations
	if !workers.IsExitWorker(id) {
		fmt.Println("No exit worker found for the given id..")
		return
	}
	if oldTarget > oldStoploss {
		if target < oldTarget || stoploss < oldStoploss {
			fmt.Println("New target/stoploss must be bigger than the last one..")
			return
		}
		ltp := utility.GetLtp(underlying)
		if oldTarget-ltp > (oldTarget-oldStoploss)*2/3 {
			fmt.Println("must book some amount of profit before raising the target..")
			return
		}
		// can't put condition on stoploss for profit booking as we can't fetch the entry here
	} else {
		if target > oldTarget || stoploss > oldStoploss {
			fmt.Println("New target/stoploss must be bigger than the last one..")
			return
		}
		ltp := utility.GetLtp(underlying)
		if ltp-oldTarget > (oldStoploss-oldTarget)*2/3 {
			fmt.Println("must book some amount of profit before raising the target..")
			return
		}
	}

	go func() {
		// cancel associated "exit" worker
		// this seems to work waiting until the worker is cancelled to avoid conditions where exitFound channel receives unwanted signals
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			workers.CancelWorker(id)
		}()
		
		// create new exit worker
		wg.Wait()
		fmt.Println("Trailing the current order..")
		// should I really use a goroutine here?
		go startExitWorker(instrument, underlying, quantity, target, stoploss)
	}()
}

// to place partial orders
func PlacePartialOrder(w http.ResponseWriter, r *http.Request) {
	// read query params
	entry, _ := strconv.ParseFloat(r.URL.Query().Get("entry"), 64)
	// target, _ := strconv.ParseFloat(r.URL.Query().Get("target"), 64)
	// stoploss, _ := strconv.ParseFloat(r.URL.Query().Get("stoploss"), 64)
	// reaffirmation, _ := strconv.ParseFloat(r.URL.Query().Get("reaffirmation"), 64)
	// newStoploss, _ := strconv.ParseFloat(r.URL.Query().Get("newStoploss"), 64)
	// newTarget, _ := strconv.ParseFloat(r.URL.Query().Get("newTarget"), 64)

	// just for testing: 
	target := entry + 10;
	stoploss := entry - 10;
	reaffirmation := entry + 5;
	newStoploss := entry - 5;
	newTarget := entry + 15;

	// fetch underlying details, quantity
	instrument_key := r.URL.Query().Get("instrument_key")
	underlying_instrument_key, quantity := getUnderlyingDetails(instrument_key, true)
	if underlying_instrument_key == "" || quantity == 0 {
		fmt.Println("Error fetching underlying details")
		return
	}

	// Validations
	if quantity == 0 {
		fmt.Println("Invalid quantity")
		return
	}
	if target > entry {
		// long order
		if target-entry < entry-stoploss {
			fmt.Println("Target/stoploss should at least be 1:1")
			return
		}
		if reaffirmation < entry {
			fmt.Println("Reaffirmation should be greater than entry")
			return
		}
		if newStoploss < stoploss {
			fmt.Println("New stoploss should be greater than stoploss")
			return
		}
		if newTarget-reaffirmation < reaffirmation-newStoploss {
			fmt.Println("New target/stoploss should at least be 1:1")
			return
		}
	} else {
		// short order
		if entry-target < stoploss-entry {
			fmt.Println("Target/stoploss should at least be 1:1")
			return
		}
		if reaffirmation > entry {
			fmt.Println("Reaffirmation should be greater than entry")
			return
		}
		if newStoploss > stoploss {
			fmt.Println("New stoploss should be greater than stoploss")
			return
		}
		if reaffirmation-newTarget < newStoploss-reaffirmation {
			fmt.Println("New target/stoploss should at least be 1:1")
			return
		}
	}

	go func() {
		// create new worker with quantity/3
		entryFound := make(chan struct{})
		entryCancelled := make(chan struct{})
		defer close(entryFound)
		defer close(entryCancelled)
		worker := workers.NewEntryWorker(uuid.New().String(), instrument_key, underlying_instrument_key, quantity, entry, entryFound, entryCancelled)
		workers.AddWorker(worker)

		// wait for entry
		select {
		case <-entryFound:
			go func() {
				fmt.Println("Entry found for the partial order!")
				// create worker for reaffirmation at stoploss
				addedAtSl := make(chan struct{})
				cancelled := make(chan struct{})
				// defer close(addedAtSl) // i think this sends value to channel addedAtSl when getting closed but block not getting executed but worker closed?
				// defer close(cancelled) to avoid panic situation
				var slEntry float64
				if target > entry {
					slEntry = entry - 0.9*(entry-stoploss)
				} else {
					slEntry = entry + 0.9*(stoploss-entry)
				}
				slId := uuid.New().String()
				slWorker := workers.NewEntryWorker(slId, instrument_key, underlying_instrument_key, quantity*2, slEntry, addedAtSl, cancelled)
				workers.AddWorker(slWorker)
	
				// create worker for reaffirmation at given point
				reAffirmId := uuid.New().String()
				addedAtReaffirm := make(chan struct{})
				// defer close(addedAtReaffirm)
				// cancelled = make(chan struct{}) if either is cancelled / timed out we exit
				reaffirmWorker := workers.NewEntryWorker(reAffirmId, instrument_key, underlying_instrument_key, quantity*2, reaffirmation, addedAtReaffirm, cancelled)
				workers.AddWorker(reaffirmWorker)
	
				select {
				case <-addedAtReaffirm:
					fmt.Println("Added at reaffirmation point..")
					// Awaiting for this would cause cancelled block to be triggered? using a wg.sync
					workers.CancelWorker(slId)
	
					// create exit worker for original target/stoploss
					go startExitWorker(instrument_key, underlying_instrument_key, quantity, target, stoploss)
	
					// create exit worker for new target/stoploss: should this really be in goroutine
					go startExitWorker(instrument_key, underlying_instrument_key, quantity*2, newTarget, newStoploss) // not absolutely necessary
	
					return // not absolutely necessary
	
				case <-addedAtSl:
					fmt.Println("Added near stoploss..")
					workers.CancelWorker(reAffirmId)
	
					// create exit worker for new target/stoploss
					go startExitWorker(instrument_key, underlying_instrument_key, quantity*3, target, stoploss)
	
					return // not absolutely necessary
	
				case <-cancelled:
					fmt.Println("One of the partial orders was timed out or cancelled..")
					// when no reaffirmation occured, need to exit for the original target/stoploss
					// this shouldn't be outside because this won't ever happen before one of the two reaffirmations
					go startExitWorker(instrument_key, underlying_instrument_key, quantity, target, stoploss)
					return // to make sure this only happens once, as ultimately there will be two cancel signals
					// but if there are two signals after cancelled is closed, there will be panic
	
				// alternatively, case timeout AS manual cancel won't work anyway:
				// 	workers.CancelWorker(slId)
				// 	workers.CancelWorker(reAffirmId)
				}
			}()
		case <-entryCancelled:
			fmt.Println("Entry Order was timed out or cancelled..")
			return
		}
	}()
	// another performance issue with above order is that we check for two entries simultaneously for same instrument
	// and again in the case of addedReaffirmation, we check for two exits in same instrument: this is inefficient: maybe raise ticker.duration?
	// solutions: or reduce timer? we could also simply call the other version implemented in dev branch here
}

// helper function for creating and starting an exitWorker to reduce code duplication
// this is blocking in nature, so use as goroutine if another code to process
func startExitWorker(instrument, underlying string, quantity int, target, stoploss float64) {
	exitFound := make(chan struct{})
	cancelled := make(chan struct{})
	exitWorker := workers.NewExitWorker(uuid.New().String(), instrument, underlying, quantity, target, stoploss, exitFound, cancelled)
	workers.AddWorker(exitWorker)
	select {
	case <-exitFound:
		fmt.Printf("Exit found for original target %f / stoploss %f..\n", target, stoploss)
	case <-cancelled:
		fmt.Printf("Exit worker for original target %f / stoploss %f was timed out or cancelled..\n", target, stoploss)
	}
	close(exitFound)
	close(cancelled)
}
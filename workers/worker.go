package workers

import (
	"context"
	"errors"
	"fmt"
	"myapp/utility"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Worker interface {
	Start(ctx context.Context) error
	Cancel()
	GetID() string
}

// Improvement: use a base struct in both EntryWorker and ExitWorker

type EntryWorker struct {
	id         string
	instrument string
	underlying string
	quantity   int
	entry      float64
	entered    chan struct{}
	cancelled  chan struct{}
	cancel     func()
}

func NewEntryWorker(id string, instrument, underlying string, quantity int, entry float64, entered, cancelled chan struct{}) *EntryWorker {
	return &EntryWorker{
		id:         id,
		instrument: instrument,
		underlying: underlying,
		quantity:   quantity,
		entry:      entry,
		entered:    entered,
		cancelled:  cancelled,
	}
}

func (w *EntryWorker) Cancel() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *EntryWorker) GetID() string {
	return w.id
}

func (w *EntryWorker) Start(ctx context.Context) error {
	fmt.Printf("EntryWorker %s started\n", w.id)
	defer fmt.Printf("EntryWorker %s completed\n", w.id)

	// Simulate work
	var ltp float64
	ltp = utility.GetLtp(w.underlying)
	if ltp == 0 {
		return nil
	}
	var checkEntryCondition func() bool
	if w.entry <= ltp {
		checkEntryCondition = func() bool {
			return ltp <= w.entry
		}
	} else {
		checkEntryCondition = func() bool {
			return ltp > w.entry
		}
	}

	ticker_duration, _ := strconv.Atoi(os.Getenv("TICKER_DURATION"))
	ticker := time.NewTicker(time.Duration(ticker_duration) * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				ltp = utility.GetLtp(w.underlying)
				fmt.Println("LTP: ", ltp, " seeking entry...")
				if checkEntryCondition() {
					fmt.Println("Entry condition met.. Placing order")
					// utility.PlaceOrder(quantity, instrument_key, "BUY")
					w.entered <- struct{}{}
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		// remove worker from workers map
		delete(workers, w.id)
		// send signal to parent
		w.cancelled <- struct{}{}
		return ctx.Err()
	case <-w.entered:
		// still need to remove worker
		delete(workers, w.id)
		return errors.New("entry found")
	}
}

type ExitWorker struct {
	id         string
	instrument string
	underlying string
	quantity   int
	target     float64
	stoploss   float64
	exited     chan struct{}
	cancelled  chan struct{}
	cancel     func()
}

func NewExitWorker(id string, instrument, underlying string, quantity int, target, stoploss float64, exited, cancelled chan struct{}) *ExitWorker {
	return &ExitWorker{
		id:         id,
		instrument: instrument,
		underlying: underlying,
		quantity:   quantity,
		target:     target,
		stoploss:   stoploss,
		exited:     exited,
		cancelled:  cancelled,
	}
}

func (w *ExitWorker) Cancel() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *ExitWorker) GetID() string {
	return w.id
}

func (w *ExitWorker) Start(ctx context.Context) error {
	fmt.Printf("ExitWorker %s started\n", w.id)
	defer fmt.Printf("ExitWorker %s completed\n", w.id)

	var ltp float64
	checkExitCondition := func() bool {
		if w.target > w.stoploss {
			return ltp >= w.target || ltp <= w.stoploss
		} else {
			return ltp <= w.target || ltp >= w.stoploss
		}
	}

	ticker_duration, _ := strconv.Atoi(os.Getenv("TICKER_DURATION"))
	ticker := time.NewTicker(time.Duration(ticker_duration) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				ltp = utility.GetLtp(w.underlying)
				fmt.Println("LTP: ", ltp, " seeking exit...")
				if checkExitCondition() {
					// Utility.PlaceOrder(quantity, instrument_key, "SELL")
					fmt.Println("Exiting position because conditions are met..")
					w.exited <- struct{}{}
					return
				}
			case <-ctx.Done():
				// Utility.PlaceOrder(quantity, instrument_key, "SELL")
				w.cancelled <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		delete(workers, w.id)
		return ctx.Err()
	case <-w.exited:
		delete(workers, w.id)
		return errors.New("exit found")
	}
}

var (
	workers      = make(map[string]Worker)
	addWorker    = make(chan Worker)
	cancelWorker = make(chan string)
	wg           sync.WaitGroup
)

func AddWorker(worker Worker) {
	addWorker <- worker
}

func CancelWorker(id string) {
	cancelWorker <- id
}

func startWorker(worker Worker) {
	defer wg.Done()
	hold_duration, _ := strconv.Atoi(os.Getenv("HOLD_DURATION"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(hold_duration)*time.Minute)
	switch w := worker.(type) {
	case *EntryWorker:
		w.cancel = cancel
	case *ExitWorker:
		w.cancel = cancel
	}
	defer cancel()
	if err := worker.Start(ctx); err != nil {
		fmt.Printf("Worker terminated %s: %v\n", worker.GetID(), err)
	}
}

func ManageWorkers() {
	for {
		select {
		case worker := <-addWorker:
			workers[worker.GetID()] = worker
			wg.Add(1) // use, wg.wait() in case you want to wait for all workers to complete
			go startWorker(worker)
		case id := <-cancelWorker:
			cancelWorkerByID(id)
		}
	}
}

func cancelWorkerByID(id string) {
	if worker, ok := workers[id]; ok {
		worker.Cancel()
		delete(workers, id)
	} else {
		fmt.Printf("Worker %s not found\n", id)
	}
}

func GetWorkerDetails(id string) (instrument, underlying string, quantity int) {
	if worker, ok := workers[id]; ok {
		if entryWorker, ok := worker.(*EntryWorker); ok {
			return entryWorker.instrument, entryWorker.underlying, entryWorker.quantity
		} else if exitWorker, ok := worker.(*ExitWorker); ok {
			return exitWorker.instrument, exitWorker.underlying, exitWorker.quantity
		}
	}
	return "", "", 0
}


func GetAllWorkers() (entryWorkers, exitWorkers string) {
	entryWorkersMap := make(map[string]Worker)
	exitWorkersMap := make(map[string]Worker)

	for id, worker := range workers {
		if entryWorker, ok := worker.(*EntryWorker); ok {
			entryWorkersMap[id] = entryWorker
		} else if exitWorker, ok := worker.(*ExitWorker); ok {
			exitWorkersMap[id] = exitWorker
		}
	}

	entryWorkers = mapToString(entryWorkersMap)
	exitWorkers = mapToString(exitWorkersMap)

	return entryWorkers, exitWorkers
}

func mapToString(m map[string]Worker) string {
	var builder strings.Builder
	builder.WriteString("{")
	for k, v := range m {
		builder.WriteString(fmt.Sprintf("%s: %s, ", k, workerToString(v)))
	}
	builder.WriteString("}")
	return builder.String()
}

func workerToString(w Worker) string {
	if entryWorker, ok := w.(*EntryWorker); ok {
		return fmt.Sprintf("{instrument: %s, underlying: %s, entry: %f}",
			entryWorker.instrument, entryWorker.underlying, entryWorker.entry)
	} else if exitWorker, ok := w.(*ExitWorker); ok {
		return fmt.Sprintf("{instrument: %s, underlying: %s, target: %f, stoploss: %f}",
			exitWorker.instrument, exitWorker.underlying, exitWorker.target, exitWorker.stoploss)
	}
	return fmt.Sprintf("{id: %s}", w.GetID())
}

func IsExitWorker(id string) bool {
	if worker, ok := workers[id]; ok {
		if _, ok := worker.(*ExitWorker); ok {
			return true
		}
	}
	return false
}
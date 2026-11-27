package controllers

import (
	"myapp/workers"
	"net/http"
)

func GetWorkers(w http.ResponseWriter, r *http.Request) {
	entryWorkers, exitWorkers := workers.GetAllWorkers()
	w.Write([]byte("Entry Workers: " + entryWorkers + "\nExit Workers: " + exitWorkers))
}

// Cancel any kind of order
func CancelOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	workers.CancelWorker(id)
}
package main

import (
	"myapp/authentication"
	"myapp/orders"
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	// Authentication
	mux.HandleFunc("/auth/gettoken", authentication.TokenHandler)
	mux.HandleFunc("/auth/logout", authentication.LogoutHandler)

	// Orders
	mux.HandleFunc("/order/placeut", orders.PlaceUTOrder)

	// Risk Management

}
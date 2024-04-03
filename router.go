package main

import (
	"myapp/authentication"
	"myapp/controllers"
	"net/http"
)

func registerRoutes(mux *http.ServeMux) {
	// Authentication
	mux.HandleFunc("/auth/gettoken", authentication.TokenHandler)
	mux.HandleFunc("/auth/logout", authentication.LogoutHandler)

	// Quote
	mux.HandleFunc("/quote/ltp", controllers.GetLTP)
	mux.HandleFunc("/quote/optionchain", controllers.GetOptionChain)

	// Orders
	mux.HandleFunc("/order/placeut", controllers.PlaceUTOrder)

	// Risk Management

}
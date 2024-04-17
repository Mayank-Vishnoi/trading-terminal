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
	mux.HandleFunc("/order/place/ut", orders.PlaceUTOrder)
	mux.HandleFunc("/order/place/paper", orders.PlaceUTPaperOrder)
	mux.HandleFunc("/order/place/trailing", orders.PlaceTrailingOrder)
	mux.HandleFunc("/order/modify/trailing", orders.ModifyTrailingOrder)
	mux.HandleFunc("/order/place/partial", orders.PlacePartialOrder)

	// Risk Management

}
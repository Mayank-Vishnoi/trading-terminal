package main

import (
	"myapp/authentication"
	"myapp/controllers"
	"net/http"
)

// func inject(ctx context.Context, f http.HandlerFunc) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		req := r.WithContext(ctx)
// 		f(w, req)
// 	}
// }

func registerRoutes(mux *http.ServeMux) {
	// Authentication
	mux.HandleFunc("/auth/gettoken", authentication.TokenHandler)
	mux.HandleFunc("/auth/logout", authentication.LogoutHandler)

	// Orders
	mux.HandleFunc("/order/place/ut", controllers.PlaceUTOrder)
	mux.HandleFunc("/order/modify", controllers.ModifyExitOrder)
	mux.HandleFunc("/order/place/partial", controllers.PlacePartialOrder)
	
	// Debug
	mux.HandleFunc("/workers/getactive", controllers.GetWorkers)
	mux.HandleFunc("/workers/cancel", controllers.CancelOrder)
}
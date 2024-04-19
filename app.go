package main

import (
	"fmt"
	"myapp/workers"
	"net/http"

	_ "github.com/joho/godotenv/autoload"
)

// Previous flow: hit the login endpoint to get the authorization code, save that code in the .env file everytime before running the server. Get the token in the main context and pass it around.
// Current flow: get auth code, hit gettoken api and save that manually in the .env, restart the server should be good for 1 day.
func main() {
	mux := http.NewServeMux()
	registerRoutes(mux)

	go workers.ManageWorkers()
	// use another goroutine to handle task completion or just cleanup in the workers itself

	fmt.Println("Server listening on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Error:", err)
	}
}
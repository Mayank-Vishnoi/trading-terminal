package main

import (
	"fmt"
	"log"
	"myapp/config"
	"myapp/workers"
	"net/http"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if err := config.Load("config.json"); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	registerRoutes(mux)

	go workers.ManageWorkers()
	// use another goroutine to handle task completion or just cleanup in the workers itself

	fmt.Println("Server listening on port 8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Error:", err)
	}
}
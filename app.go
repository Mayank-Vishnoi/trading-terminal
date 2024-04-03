package main

import (
	"fmt"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Application have started!")
}

func main() {
	// Initialize a new session manager and configure the session lifetime

	// Create a new router
	mux := http.NewServeMux()

	mux.HandleFunc("/", handler)

	// Register other routes
	registerRoutes(mux)

	// Start the HTTP server on port 8080
	fmt.Println("Server listening on port 8080...")
	// Wrap your handlers with the LoadAndSave() middleware
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Error:", err)
	}

}
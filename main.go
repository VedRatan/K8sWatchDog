package main

import (
	"log"
	"net/http"
	"time"

	"github.com/VedRatan/k8s-agent/handlers"
	"github.com/gorilla/mux"
)

func startServer(router *mux.Router) {
	server := &http.Server{
		Addr:           ":8080",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // Set max header size (e.g., 1 MB)
	}

	log.Printf("Server is starting at :8080")
	// Start the server
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/apply", handlers.ApplyHandler).Methods("POST")

	log.Println("Starting k8s-agent on :8080")
	startServer(r)
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/VedRatan/k8s-agent/handlers"
	"github.com/gorilla/mux"
)

func startServer(router *mux.Router) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
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
	r.HandleFunc("/pods", handlers.ListPodsHandler).Methods("GET")
	r.HandleFunc("/pods/{namespace}/{podName}/logs", handlers.StreamLogsHandler).Methods("GET")
	r.HandleFunc("/pods/{namespace}/{podName}/status", handlers.PodStatusHandler).Methods("GET")
	log.Println("Starting k8s-agent on :8080")
	startServer(r)
}

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/VedRatan/remediation-server/handlers"
	"github.com/VedRatan/remediation-server/types"
	"github.com/gorilla/mux"
)

func startServer(router *mux.Router) {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "7070" // default port
	}
	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // Set max header size (e.g., 1 MB)
	}

	log.Printf("Remediation server is starting at :%s", port)
	// Start the server
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func main() {
	flag.StringVar(&types.K8sAgentServiceURL, "k8s-agent-url", "", "The LoadBalancer IP or DNS of the k8s-agent-service (required)")
	flag.Parse()

	// Validate the flag
	if types.K8sAgentServiceURL == "" {
		fmt.Println("Error: The --k8s-agent-url flag is required")
		flag.Usage()
		os.Exit(1) // Exit with a non-zero status code
	}
	r := mux.NewRouter()
	// Define the HTTP server
	r.HandleFunc("/webhook", handlers.WebhookHandler)

	startServer(r)
}

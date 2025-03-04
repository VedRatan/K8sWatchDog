package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var clientset *kubernetes.Clientset

func init() { //nolint:gochecknoinits
	// Load incluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig file if app is ran locally
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}
}

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

func ApplyHandler(w http.ResponseWriter, r *http.Request) {
	manifest, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var podManifest corev1.Pod
	err = yaml.Unmarshal(manifest, &podManifest)
	if err != nil {
		http.Error(w, "Failed to decode YAML manifest into Pod", http.StatusBadRequest)
		return
	}

	namespace := podManifest.Namespace
	if namespace == "" {
		namespace = "default" // if namespace field is empty, default namespace should be used
	}

	_, err = clientset.CoreV1().Pods(namespace).Create(context.Background(), &podManifest, v1.CreateOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply manifest: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"message": "Pod manifest applied successfully"}`))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/apply", ApplyHandler).Methods("POST")

	log.Println("Starting k8s-agent on :8080")
	startServer(r)
}

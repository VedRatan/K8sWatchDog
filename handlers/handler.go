package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

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

func ListPodsHandler(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {	
		namespace = "default"
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(pods.Items)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

func StreamLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	podName := vars["podName"]

	logs, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{}).Stream(context.TODO())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stream logs: %v", err), http.StatusInternalServerError)
		return
	}
	defer logs.Close()

	w.Header().Set("Content-Type", "text/plain")
	_, err = io.Copy(w, logs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

func PodStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	podName := vars["podName"]

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, v1.GetOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pod status: %v", err), http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"phase":      pod.Status.Phase,
		"conditions": pod.Status.Conditions,
	}

	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	customlogger "github.com/VedRatan/k8swatchdog/logger"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	ERROR_RESPONSE     = "Failed to write response: %v"
	LOG_ERROR_RESPONSE = "failed to write response"
)

var (
	clientset *kubernetes.Clientset
	logger    *zap.Logger
)

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

	logger, err = customlogger.NewLogger("k8s-agent")
	if err != nil {
		log.Fatalf("Error initializing logger: %v", err)
	}
}

func ApplyHandler(w http.ResponseWriter, r *http.Request) {
	checkForPodDelete := false
	manifest, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var podManifest corev1.Pod
	err = yaml.Unmarshal(manifest, &podManifest)
	if err != nil {
		logger.Error("Error occurred", zap.Error(err))
		http.Error(w, "Failed to decode YAML manifest into Pod", http.StatusBadRequest)
		return
	}

	namespace := podManifest.Namespace
	if namespace == "" {
		namespace = "default" // if namespace field is empty, default namespace should be used
	}
	name := podManifest.Name
	err = clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, v1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error("Failed to delete pod", zap.Error(err))
	} else if err == nil {
		checkForPodDelete = true
	}

	if checkForPodDelete {
		watcher, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), v1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create watcher %v", err), http.StatusInternalServerError)
		}
		defer watcher.Stop()

	Loop:
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Deleted:
				logger.Info("pod has been deleted", zap.String("name", name))
				watcher.Stop() // graceful shutdown
				break Loop
			case watch.Error:
				logger.Error("error watching pod", zap.String("name", name))
				http.Error(w, fmt.Sprintf("error watching pod: %v", event.Object), http.StatusInternalServerError)
				return
			}
		}
	}

	_, err = clientset.CoreV1().Pods(namespace).Create(context.Background(), &podManifest, v1.CreateOptions{})
	if err != nil {
		logger.Error("failed to apply manifest", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to apply manifest: %v", err), http.StatusInternalServerError)
		return
	}
	logger.Info("remediated pod has been created", zap.String("name", namespace+"/"+name))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"message": "Pod manifest applied successfully"}`))
	if err != nil {
		logger.Error(LOG_ERROR_RESPONSE, zap.Error(err))
		http.Error(w, fmt.Sprintf(ERROR_RESPONSE, err), http.StatusInternalServerError)
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
		logger.Error("failed to list pods", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(pods.Items)
	if err != nil {
		logger.Error(LOG_ERROR_RESPONSE, zap.Error(err))
		http.Error(w, fmt.Sprintf(ERROR_RESPONSE, err), http.StatusInternalServerError)
		return
	}
}

func StreamLogsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	podName := vars["podName"]

	logs, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{}).Stream(context.TODO())
	if err != nil {
		logger.Error("failed to get logs", zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to stream logs: %v", err), http.StatusInternalServerError)
		return
	}
	defer logs.Close()

	w.Header().Set("Content-Type", "text/plain")
	_, err = io.Copy(w, logs)
	if err != nil {
		logger.Error(LOG_ERROR_RESPONSE, zap.Error(err))
		http.Error(w, fmt.Sprintf(ERROR_RESPONSE, err), http.StatusInternalServerError)
		return
	}
}

func PodStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	podName := vars["podName"]

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, v1.GetOptions{})
	if err != nil {
		log.Println("failed to get the pod status", err.Error())
		http.Error(w, fmt.Sprintf("Failed to get pod status: %v", err), http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"phase":      pod.Status.Phase,
		"conditions": pod.Status.Conditions,
	}

	err = json.NewEncoder(w).Encode(status)
	if err != nil {
		log.Println("failed to write response", err.Error())
		http.Error(w, fmt.Sprintf(ERROR_RESPONSE, err), http.StatusInternalServerError)
		return
	}
}

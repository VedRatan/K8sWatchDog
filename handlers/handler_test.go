package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAll(t *testing.T) {
	// Run tests in sequence
	t.Run("TestApplyHandler", testApplyHandler)
	t.Run("TestListPodsHandler", testListPodsHandler)
	t.Run("TestStreamLogsHandler", testStreamLogsHandler)
	t.Run("TestPodStatusHandler", testPodStatusHandler)
	t.Cleanup(cleanupFunction)
}

func testApplyHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), "POST", "/apply", strings.NewReader(`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
spec:
  containers:
  - name: test-container
    image: nginx`))
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ApplyHandler)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Pod manifest applied successfully")
}

func testPodStatusHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), "GET", "/pods/default/test-pod/status", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/pods/{namespace}/{podName}/status", PodStatusHandler)

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func testListPodsHandler(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), "GET", "/pods?namespace=default", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ListPodsHandler)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func testStreamLogsHandler(t *testing.T) {
	time.Sleep(5 * time.Second)
	req, err := http.NewRequestWithContext(t.Context(), "GET", "/pods/default/test-pod/logs", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/pods/{namespace}/{podName}/logs", StreamLogsHandler)

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code) // Pod does not exist
}

func cleanupFunction() {
	err := clientset.CoreV1().Pods("default").Delete(context.Background(), "test-pod", v1.DeleteOptions{})
	if err != nil {
		fmt.Println("Failed to delete test pod:", err.Error())
		return
	}
}

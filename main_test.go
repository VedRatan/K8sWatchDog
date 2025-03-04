package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAll(t *testing.T) {
	// Run tests in sequence
	t.Run("TestApplyHandler", TestApplyHandler)
	t.Cleanup(cleanupFunction)
}

func TestApplyHandler(t *testing.T) {
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

func cleanupFunction() {
	err := clientset.CoreV1().Pods("default").Delete(context.Background(), "test-pod", v1.DeleteOptions{})
	if err != nil {
		fmt.Println("Failed to delete test pod:", err.Error())
		return
	}
}

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/VedRatan/remediation-server/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

// function to  extract pod name and namespace from the provided remediationYAML
func ExtractPodDetails(remediationYAML string) (string, string, error) {
	obj := &unstructured.Unstructured{}
	decoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := decoder.Decode([]byte(remediationYAML), nil, obj)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode remediation YAML: %v", err)
	}

	podName := obj.GetName()
	namespace := obj.GetNamespace()
	if podName == "" || namespace == "" {
		return "", "", fmt.Errorf("pod name or namespace not found in the remediation YAML")
	}

	return podName, namespace, nil
}

func ApplyRemediation(remediationYAML string) error {
	var url string
	// conditional url building based on the type of connection between remediation-server and k8s-agent
	if types.Insecure {
		url = fmt.Sprintf("http://%s/apply", types.K8sAgentServiceURL)
	} else {
		url = fmt.Sprintf("https://%s/apply", types.K8sAgentServiceURL)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(remediationYAML))
	if err != nil {
		return fmt.Errorf("Error creating POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send remediation YAML to k8s-agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("k8s-agent returned non-OK status: %s | response: %s", resp.Status, string(bodyBytes))
	}

	return nil
}

func VerifyPodStatus(namespace, podName string, isRemediated bool) error {
	statusURL := fmt.Sprintf("http://%s/pods/%s/%s/status", types.K8sAgentServiceURL, namespace, podName)
	// If the pod is remediated, it will take some time to comeup in a ready state
	if isRemediated {
		for i := 0; i < 5; i++ { // Retry 5 times with a delay
			time.Sleep(10 * time.Second) // Wait for 10 seconds before checking the status
			status, err := getPodStatus(statusURL)
			if err != nil {
				return err
			}
			// Check if the pod is in the Ready state
			if isPodReady(status) {
				return nil
			}
		}
	} else {
		status, err := getPodStatus(statusURL)
		if err != nil {
			return err
		}
		if isPodReady(status) {
			return nil
		}
	}

	return fmt.Errorf("pod %s/%s did not reach Ready state within the timeout period", namespace, podName)
}

func getPodStatus(statusURL string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating GET request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check pod status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("k8s-agent returned non-OK status: %s", resp.Status)
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode pod status: %v", err)
	}
	return status, nil
}

func isPodReady(status map[string]interface{}) bool {
	if phase, ok := status["phase"].(string); !ok || (phase != "Running" && phase != "Succeeded") { // In case the pod is part of a job
		return false
	}

	// If the pod has succeeded, it is considered "ready" (exited successfully)
	if status["phase"].(string) == "Succeeded" {
		return true
	}

	// Check if the "Ready" condition is true
	if conditions, ok := status["conditions"].([]interface{}); ok {
		for _, condition := range conditions {
			if cond, ok := condition.(map[string]interface{}); ok {
				if cond["type"].(string) == "Ready" && cond["status"].(string) == "True" {
					return true
				}
			}
		}
	}

	return false
}

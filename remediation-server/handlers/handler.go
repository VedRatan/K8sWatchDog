package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/VedRatan/remediation-server/types"
)

// Function to send the remediated YAML to k8s-agent service
func ForwardRemediation(remediationYAML string) error {
	podName, namespace, err := extractPodDetails(remediationYAML)
	if err != nil {
		return fmt.Errorf("failed to extract pod details: %v", err)
	}

	// Apply the remediation YAML via k8s-agent service
	if err := applyRemediation(remediationYAML); err != nil {
		return fmt.Errorf("failed to apply remediation: %v", err)
	}

	// Verify the pod status
	if err := verifyPodStatus(namespace, podName); err != nil {
		return fmt.Errorf("failed to verify pod status: %v", err)
	}

	return nil
}

// To make remediation server work as a server as well, we added this function. TODO: Utilize this function if k8sagent supports connecting to third party services in near future.
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var alert types.Alert
	if err := json.Unmarshal(body, &alert); err != nil {
		http.Error(w, "Failed to parse alert", http.StatusBadRequest)
		return
	}

	// Forward the remediation YAML to the k8s-agent service
	if err := ForwardRemediation(alert.RemediationYAML); err != nil {
		http.Error(w, fmt.Sprintf("Failed to forward remediation: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Remediation applied successfully and pod is in Ready state"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

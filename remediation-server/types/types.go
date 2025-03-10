package types

var (
	K8sAgentServiceURL string // Flag to store the k8s-agent-service LoadBalancer IP
)

// Alert struct with the expected parameters
type Alert struct {
	Description     string `json:"description"`
	RemediationYAML string `json:"remediationYAML"`
}

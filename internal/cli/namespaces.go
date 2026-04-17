package cli

import "fmt"

// protectedNamespaces blocks chaos experiments against Kubernetes system
// namespaces. Non-negotiable — these are hard-coded constraints, not
// configuration.
var protectedNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

// ensureNamespaceAllowed returns an error when the namespace is protected.
func ensureNamespaceAllowed(namespace string) error {
	if protectedNamespaces[namespace] {
		return fmt.Errorf("namespace %q is protected — chaos experiments are not allowed in system namespaces", namespace)
	}

	return nil
}

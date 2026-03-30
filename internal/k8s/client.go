package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps a Kubernetes clientset.
type Client struct {
	clientset kubernetes.Interface
}

// NewClient builds a Client from a kubeconfig path.
// Falls back to in-cluster config when kubeconfig is empty.
func NewClient(kubeconfig string) (*Client, error) {
	config, err := buildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("building k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating k8s clientset: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

// GetPods lists pods matching a label selector in a namespace.
func (c *Client) GetPods(ctx context.Context, namespace, selector string) ([]corev1.Pod, error) {
	list, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	return list.Items, nil
}

// DeletePod deletes a single pod with the given grace period in seconds.
func (c *Client) DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error {
	err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})
	if err != nil {
		return fmt.Errorf("deleting pod %s/%s: %w", namespace, name, err)
	}

	return nil
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return rest.InClusterConfig()
}

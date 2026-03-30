package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// PodManager abstracts pod operations for testability.
type PodManager interface {
	GetPods(ctx context.Context, namespace, selector string) ([]corev1.Pod, error)
	DeletePod(ctx context.Context, namespace, name string, gracePeriod int64) error
}

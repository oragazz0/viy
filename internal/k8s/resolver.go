package k8s

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

// ResolvedTarget holds the result of target resolution.
type ResolvedTarget struct {
	ResourceFound bool
	ResourceKind  string
	ResourceName  string
	Selector      string
	Pods          []corev1.Pod
}

// TargetResolver resolves an eyes.Target into concrete pods.
type TargetResolver interface {
	Resolve(ctx context.Context, target eyes.Target) (*ResolvedTarget, error)
}

// Resolver resolves targets by querying the Kubernetes API.
type Resolver struct {
	client *Client
}

// NewResolver creates a Resolver backed by the given Client.
func NewResolver(client *Client) *Resolver {
	return &Resolver{client: client}
}

// Resolve looks up the target resource, extracts its pod selector,
// merges any user-supplied selector, and returns matching pods.
func (r *Resolver) Resolve(ctx context.Context, target eyes.Target) (*ResolvedTarget, error) {
	kind := normalizeKind(target.Kind)

	resourceSelector, err := r.extractResourceSelector(ctx, kind, target.Namespace, target.Name)
	if err != nil {
		return nil, err
	}

	merged := mergeSelectors(resourceSelector, target.Selector)

	pods, err := r.client.GetPods(ctx, target.Namespace, merged)
	if err != nil {
		return nil, fmt.Errorf("listing pods for %s/%s: %w", target.Namespace, target.Name, err)
	}

	return &ResolvedTarget{
		ResourceFound: true,
		ResourceKind:  kind,
		ResourceName:  target.Name,
		Selector:      merged,
		Pods:          pods,
	}, nil
}

func (r *Resolver) extractResourceSelector(ctx context.Context, kind, namespace, name string) (string, error) {
	switch kind {
	case "pod":
		return r.resolvePodSelector(ctx, namespace, name)
	case "deployment":
		return r.resolveDeploymentSelector(ctx, namespace, name)
	case "statefulset":
		return r.resolveStatefulSetSelector(ctx, namespace, name)
	case "service":
		return r.resolveServiceSelector(ctx, namespace, name)
	default:
		return "", fmt.Errorf("%w: %s (supported: pod, deployment, statefulset, service)",
			viyerrors.ErrUnsupportedResourceKind, kind)
	}
}

func (r *Resolver) resolvePodSelector(ctx context.Context, namespace, name string) (string, error) {
	pod, err := r.client.GetPod(ctx, namespace, name)
	if err != nil {
		return "", fmt.Errorf("%w: pod %s/%s", viyerrors.ErrTargetNotFound, namespace, name)
	}

	return metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: pod.Labels,
	}), nil
}

func (r *Resolver) resolveDeploymentSelector(ctx context.Context, namespace, name string) (string, error) {
	deployment, err := r.client.GetDeployment(ctx, namespace, name)
	if err != nil {
		return "", fmt.Errorf("%w: deployment %s/%s", viyerrors.ErrTargetNotFound, namespace, name)
	}

	return metav1.FormatLabelSelector(deployment.Spec.Selector), nil
}

func (r *Resolver) resolveStatefulSetSelector(ctx context.Context, namespace, name string) (string, error) {
	statefulSet, err := r.client.GetStatefulSet(ctx, namespace, name)
	if err != nil {
		return "", fmt.Errorf("%w: statefulset %s/%s", viyerrors.ErrTargetNotFound, namespace, name)
	}

	return metav1.FormatLabelSelector(statefulSet.Spec.Selector), nil
}

func (r *Resolver) resolveServiceSelector(ctx context.Context, namespace, name string) (string, error) {
	service, err := r.client.GetService(ctx, namespace, name)
	if err != nil {
		return "", fmt.Errorf("%w: service %s/%s", viyerrors.ErrTargetNotFound, namespace, name)
	}

	if len(service.Spec.Selector) == 0 {
		return "", fmt.Errorf("%w: service %s/%s has no pod selector",
			viyerrors.ErrTargetNotFound, namespace, name)
	}

	return metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: service.Spec.Selector,
	}), nil
}

func normalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

// mergeSelectors combines two comma-separated label selectors.
// Either or both may be empty.
func mergeSelectors(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)

	if a == "" {
		return b
	}

	if b == "" {
		return a
	}

	return a + "," + b
}

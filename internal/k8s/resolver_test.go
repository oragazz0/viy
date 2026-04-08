package k8s

import (
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

func newTestClient(objects ...runtime.Object) *Client {
	return &Client{clientset: fake.NewSimpleClientset(objects...)}
}

func TestResolve_Deployment(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			},
		},
	}

	pods := []runtime.Object{
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-abc",
				Namespace: "default",
				Labels:    map[string]string{"app": "api"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-def",
				Namespace: "default",
				Labels:    map[string]string{"app": "api"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-xyz",
				Namespace: "default",
				Labels:    map[string]string{"app": "web"},
			},
		},
	}

	objects := append([]runtime.Object{deployment}, pods...)
	resolver := NewResolver(newTestClient(objects...))

	result, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Deployment",
		Name:      "api-server",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if !result.ResourceFound {
		t.Error("ResourceFound should be true")
	}

	if len(result.Pods) != 2 {
		t.Errorf("Resolve() returned %d pods, want 2", len(result.Pods))
	}

	if result.Selector != "app=api" {
		t.Errorf("Selector = %q, want %q", result.Selector, "app=api")
	}
}

func TestResolve_StatefulSet(t *testing.T) {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "database",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "db"},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "database-0",
			Namespace: "default",
			Labels:    map[string]string{"app": "db"},
		},
	}

	resolver := NewResolver(newTestClient(statefulSet, pod))

	result, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "StatefulSet",
		Name:      "database",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(result.Pods) != 1 {
		t.Errorf("Resolve() returned %d pods, want 1", len(result.Pods))
	}

	if result.ResourceKind != "statefulset" {
		t.Errorf("ResourceKind = %q, want %q", result.ResourceKind, "statefulset")
	}
}

func TestResolve_Service(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "api"},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-abc",
			Namespace: "default",
			Labels:    map[string]string{"app": "api"},
		},
	}

	resolver := NewResolver(newTestClient(service, pod))

	result, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Service",
		Name:      "api",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(result.Pods) != 1 {
		t.Errorf("Resolve() returned %d pods, want 1", len(result.Pods))
	}
}

func TestResolve_ServiceWithoutSelector(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{},
	}

	resolver := NewResolver(newTestClient(service))

	_, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Service",
		Name:      "external",
		Namespace: "default",
	})
	if err == nil {
		t.Fatal("Resolve() should fail for service without selector")
	}

	if !errors.Is(err, viyerrors.ErrTargetNotFound) {
		t.Errorf("error should wrap ErrTargetNotFound, got: %v", err)
	}
}

func TestResolve_Pod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-abc",
			Namespace: "default",
			Labels:    map[string]string{"app": "api", "version": "v2"},
		},
	}

	matchingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-def",
			Namespace: "default",
			Labels:    map[string]string{"app": "api", "version": "v2"},
		},
	}

	resolver := NewResolver(newTestClient(pod, matchingPod))

	result, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Pod",
		Name:      "api-abc",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if result.ResourceKind != "pod" {
		t.Errorf("ResourceKind = %q, want %q", result.ResourceKind, "pod")
	}

	// Should match both pods since they share the same labels.
	if len(result.Pods) != 2 {
		t.Errorf("Resolve() returned %d pods, want 2", len(result.Pods))
	}
}

func TestResolve_DeploymentNotFound(t *testing.T) {
	resolver := NewResolver(newTestClient())

	_, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Deployment",
		Name:      "nonexistent",
		Namespace: "default",
	})
	if err == nil {
		t.Fatal("Resolve() should fail for nonexistent deployment")
	}

	if !errors.Is(err, viyerrors.ErrTargetNotFound) {
		t.Errorf("error should wrap ErrTargetNotFound, got: %v", err)
	}
}

func TestResolve_UnsupportedKind(t *testing.T) {
	resolver := NewResolver(newTestClient())

	_, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "DaemonSet",
		Name:      "something",
		Namespace: "default",
	})
	if err == nil {
		t.Fatal("Resolve() should fail for unsupported kind")
	}

	if !errors.Is(err, viyerrors.ErrUnsupportedResourceKind) {
		t.Errorf("error should wrap ErrUnsupportedResourceKind, got: %v", err)
	}
}

func TestResolve_MergesUserSelector(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			},
		},
	}

	v2Pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-v2",
			Namespace: "default",
			Labels:    map[string]string{"app": "api", "version": "v2"},
		},
	}

	v1Pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-v1",
			Namespace: "default",
			Labels:    map[string]string{"app": "api", "version": "v1"},
		},
	}

	resolver := NewResolver(newTestClient(deployment, v2Pod, v1Pod))

	result, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Deployment",
		Name:      "api-server",
		Namespace: "default",
		Selector:  "version=v2",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if len(result.Pods) != 1 {
		t.Errorf("Resolve() returned %d pods, want 1 (filtered by version=v2)", len(result.Pods))
	}

	if len(result.Pods) == 1 && result.Pods[0].Name != "api-v2" {
		t.Errorf("expected pod api-v2, got %s", result.Pods[0].Name)
	}
}

func TestResolve_NoPods(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			},
		},
	}

	resolver := NewResolver(newTestClient(deployment))

	result, err := resolver.Resolve(context.Background(), eyes.Target{
		Kind:      "Deployment",
		Name:      "api-server",
		Namespace: "default",
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if !result.ResourceFound {
		t.Error("ResourceFound should be true even with no pods")
	}

	if len(result.Pods) != 0 {
		t.Errorf("Resolve() returned %d pods, want 0", len(result.Pods))
	}
}

func TestResolve_KindNormalization(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "api"},
			},
		},
	}

	resolver := NewResolver(newTestClient(deployment))

	kinds := []string{"Deployment", "deployment", "DEPLOYMENT", " deployment "}
	for _, kind := range kinds {
		result, err := resolver.Resolve(context.Background(), eyes.Target{
			Kind:      kind,
			Name:      "api",
			Namespace: "default",
		})
		if err != nil {
			t.Errorf("Resolve(kind=%q) error = %v", kind, err)
			continue
		}

		if result.ResourceKind != "deployment" {
			t.Errorf("Resolve(kind=%q) ResourceKind = %q, want %q", kind, result.ResourceKind, "deployment")
		}
	}
}

func TestMergeSelectors(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{name: "both empty", a: "", b: "", want: ""},
		{name: "a only", a: "app=api", b: "", want: "app=api"},
		{name: "b only", a: "", b: "version=v2", want: "version=v2"},
		{name: "both set", a: "app=api", b: "version=v2", want: "app=api,version=v2"},
		{name: "whitespace trimmed", a: " app=api ", b: " version=v2 ", want: "app=api,version=v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeSelectors(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("mergeSelectors(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

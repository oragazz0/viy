package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPods_ReturnsMatchingPods(t *testing.T) {
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

	client := &Client{clientset: fake.NewSimpleClientset(pods...)}

	result, err := client.GetPods(context.Background(), "default", "app=api")
	if err != nil {
		t.Fatalf("GetPods() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("GetPods() returned %d pods, want 2", len(result))
	}
}

func TestGetPods_EmptyNamespace(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset()}

	result, err := client.GetPods(context.Background(), "default", "app=api")
	if err != nil {
		t.Fatalf("GetPods() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("GetPods() returned %d pods, want 0", len(result))
	}
}

func TestDeletePod_Success(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-abc",
			Namespace: "default",
		},
	}

	client := &Client{clientset: fake.NewSimpleClientset(pod)}

	err := client.DeletePod(context.Background(), "default", "api-abc", 30)
	if err != nil {
		t.Fatalf("DeletePod() error = %v", err)
	}

	remaining, _ := client.GetPods(context.Background(), "default", "")
	if len(remaining) != 0 {
		t.Error("pod should have been deleted")
	}
}

func TestDeletePod_NotFound(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset()}

	err := client.DeletePod(context.Background(), "default", "nonexistent", 30)
	if err == nil {
		t.Fatal("DeletePod() should fail for nonexistent pod")
	}
}

func TestBuildConfig_WithKubeconfig(t *testing.T) {
	_, err := buildConfig("/nonexistent/kubeconfig")
	if err == nil {
		t.Fatal("buildConfig() should fail with nonexistent kubeconfig path")
	}
}

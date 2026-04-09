package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetDeployment_Found(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
		},
	}

	client := &Client{clientset: fake.NewSimpleClientset(deployment)}

	result, err := client.GetDeployment(context.Background(), "default", "api-server")
	if err != nil {
		t.Fatalf("GetDeployment() error = %v", err)
	}

	if result.Name != "api-server" {
		t.Errorf("GetDeployment() name = %q, want %q", result.Name, "api-server")
	}
}

func TestGetDeployment_NotFound(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset()}

	_, err := client.GetDeployment(context.Background(), "default", "nonexistent")
	if err == nil {
		t.Fatal("GetDeployment() should fail for nonexistent deployment")
	}
}

func TestGetStatefulSet_Found(t *testing.T) {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "database",
			Namespace: "default",
		},
	}

	client := &Client{clientset: fake.NewSimpleClientset(statefulSet)}

	result, err := client.GetStatefulSet(context.Background(), "default", "database")
	if err != nil {
		t.Fatalf("GetStatefulSet() error = %v", err)
	}

	if result.Name != "database" {
		t.Errorf("GetStatefulSet() name = %q, want %q", result.Name, "database")
	}
}

func TestGetStatefulSet_NotFound(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset()}

	_, err := client.GetStatefulSet(context.Background(), "default", "nonexistent")
	if err == nil {
		t.Fatal("GetStatefulSet() should fail for nonexistent statefulset")
	}
}

func TestGetService_Found(t *testing.T) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "api"},
		},
	}

	client := &Client{clientset: fake.NewSimpleClientset(service)}

	result, err := client.GetService(context.Background(), "default", "api")
	if err != nil {
		t.Fatalf("GetService() error = %v", err)
	}

	if result.Spec.Selector["app"] != "api" {
		t.Errorf("GetService() selector = %v, want app=api", result.Spec.Selector)
	}
}

func TestGetService_NotFound(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset()}

	_, err := client.GetService(context.Background(), "default", "nonexistent")
	if err == nil {
		t.Fatal("GetService() should fail for nonexistent service")
	}
}

func TestGetPod_Found(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-abc",
			Namespace: "default",
			Labels:    map[string]string{"app": "api"},
		},
	}

	client := &Client{clientset: fake.NewSimpleClientset(pod)}

	result, err := client.GetPod(context.Background(), "default", "api-abc")
	if err != nil {
		t.Fatalf("GetPod() error = %v", err)
	}

	if result.Name != "api-abc" {
		t.Errorf("GetPod() name = %q, want %q", result.Name, "api-abc")
	}
}

func TestGetPod_NotFound(t *testing.T) {
	client := &Client{clientset: fake.NewSimpleClientset()}

	_, err := client.GetPod(context.Background(), "default", "nonexistent")
	if err == nil {
		t.Fatal("GetPod() should fail for nonexistent pod")
	}
}

func TestGetPod_WrongNamespace(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-abc",
			Namespace: "production",
		},
	}

	objects := []runtime.Object{pod}
	client := &Client{clientset: fake.NewSimpleClientset(objects...)}

	_, err := client.GetPod(context.Background(), "default", "api-abc")
	if err == nil {
		t.Fatal("GetPod() should fail when pod is in a different namespace")
	}
}

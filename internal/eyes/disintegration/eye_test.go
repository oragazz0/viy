package disintegration

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
	"github.com/oragazz0/viy/pkg/eyes/eyestest"
)

// --- mock PodManager ---

type mockPodManager struct {
	pods      []corev1.Pod
	getPodErr error
	deleteErr error
	deleted   []string
}

func (m *mockPodManager) GetPods(_ context.Context, _, _ string) ([]corev1.Pod, error) {
	return m.pods, m.getPodErr
}

func (m *mockPodManager) DeletePod(_ context.Context, _, name string, _ int64) error {
	m.deleted = append(m.deleted, name)
	return m.deleteErr
}

// --- helpers ---

func makePods(names ...string) []corev1.Pod {
	pods := make([]corev1.Pod, len(names))
	for index, name := range names {
		pods[index] = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
		}
	}

	return pods
}

func newTestEye(manager *mockPodManager) *Eye {
	logger, _ := zap.NewDevelopment()
	return &Eye{
		podManager: manager,
		logger:     logger,
	}
}

func testTarget() eyes.Target {
	return eyes.Target{
		Kind:      "Deployment",
		Name:      "api",
		Namespace: "default",
		Selector:  "app=api",
	}
}

// --- contract tests ---

func TestContract(t *testing.T) {
	factory := func(deps eyes.Dependencies) eyes.Eye {
		return &Eye{
			podManager: deps.PodManager,
			logger:     deps.Logger,
		}
	}

	validConfig := &Config{PodKillCount: 1, Strategy: "random"}
	invalidConfig := &Config{}

	eyestest.RunContractTests(t, factory, validConfig, invalidConfig)
}

// --- tests ---

func TestEye_Unveil_KillCount(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b", "pod-c", "pod-d", "pod-e"),
	}

	eye := newTestEye(manager)
	config := &Config{PodKillCount: 3, Strategy: "sequential"}

	err := eye.Unveil(context.Background(), testTarget(), config)
	if err != nil {
		t.Fatalf("Unveil() error = %v", err)
	}

	if len(manager.deleted) != 3 {
		t.Errorf("deleted %d pods, want 3", len(manager.deleted))
	}
}

func TestEye_Unveil_KillPercentage(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b", "pod-c", "pod-d", "pod-e",
			"pod-f", "pod-g", "pod-h", "pod-i", "pod-j"),
	}

	eye := newTestEye(manager)
	config := &Config{PodKillPercentage: 30, Strategy: "sequential"}

	err := eye.Unveil(context.Background(), testTarget(), config)
	if err != nil {
		t.Fatalf("Unveil() error = %v", err)
	}

	if len(manager.deleted) != 3 {
		t.Errorf("deleted %d pods, want 3 (30%% of 10)", len(manager.deleted))
	}
}

func TestEye_Unveil_InsufficientPods(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b"),
	}

	eye := newTestEye(manager)
	config := &Config{PodKillCount: 5, Strategy: "sequential"}

	err := eye.Unveil(context.Background(), testTarget(), config)
	if err == nil {
		t.Fatal("Unveil() should fail with insufficient pods")
	}

	if !errors.Is(err, viyerrors.ErrInsufficientTargets) {
		t.Errorf("error = %v, want ErrInsufficientTargets", err)
	}
}

func TestEye_Unveil_GetPodsError(t *testing.T) {
	manager := &mockPodManager{
		getPodErr: errors.New("connection refused"),
	}

	eye := newTestEye(manager)
	config := &Config{PodKillCount: 1}

	err := eye.Unveil(context.Background(), testTarget(), config)
	if err == nil {
		t.Fatal("Unveil() should propagate GetPods error")
	}
}

func TestEye_Unveil_DeleteError(t *testing.T) {
	manager := &mockPodManager{
		pods:      makePods("pod-a"),
		deleteErr: errors.New("forbidden"),
	}

	eye := newTestEye(manager)
	config := &Config{PodKillCount: 1}

	err := eye.Unveil(context.Background(), testTarget(), config)
	if err == nil {
		t.Fatal("Unveil() should propagate DeletePod error")
	}

	metrics := eye.Observe()
	if metrics.ErrorsTotal != 1 {
		t.Errorf("ErrorsTotal = %d, want 1", metrics.ErrorsTotal)
	}
}

func TestEye_Unveil_ContextCancellation(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b", "pod-c"),
	}

	eye := newTestEye(manager)
	config := &Config{PodKillCount: 3, Strategy: "sequential", Interval: 1 * time.Hour}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := eye.Unveil(ctx, testTarget(), config)

	if len(manager.deleted) != 1 {
		t.Errorf("deleted %d pods, want 1 (cancelled after first)", len(manager.deleted))
	}

	if err == nil {
		t.Error("Unveil() should return context error")
	}
}

func TestEye_Observe_ActiveState(t *testing.T) {
	eye := &Eye{}

	metrics := eye.Observe()
	if metrics.IsActive {
		t.Error("new eye should not be active")
	}
}

func TestEye_Observe_ReturnsEyeName(t *testing.T) {
	eye := &Eye{}

	metrics := eye.Observe()
	if metrics.EyeName != "disintegration" {
		t.Errorf("Observe().EyeName = %q, want %q", metrics.EyeName, "disintegration")
	}
}

func TestEye_Name(t *testing.T) {
	eye := &Eye{}
	if eye.Name() != "disintegration" {
		t.Errorf("Name() = %q, want %q", eye.Name(), "disintegration")
	}
}

func TestEye_Description(t *testing.T) {
	eye := &Eye{}
	if eye.Description() == "" {
		t.Error("Description() should not be empty")
	}
}

func TestEye_Validate_ValidConfig(t *testing.T) {
	eye := &Eye{}
	config := &Config{PodKillCount: 1, Strategy: "random"}

	if err := eye.Validate(config); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestEye_Validate_InvalidConfig(t *testing.T) {
	eye := &Eye{}
	config := &Config{}

	err := eye.Validate(config)
	if err == nil {
		t.Fatal("Validate() should fail with empty config")
	}
}

func TestEye_Validate_WrongConfigType(t *testing.T) {
	eye := &Eye{}

	err := eye.Validate(wrongConfig{})
	if err == nil {
		t.Fatal("Validate() should fail with wrong config type")
	}
}

func TestEye_Pause(t *testing.T) {
	eye := &Eye{}
	eye.active.Store(true)

	if err := eye.Pause(context.Background()); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	if eye.active.Load() {
		t.Error("Pause() should set active to false")
	}
}

func TestEye_Close(t *testing.T) {
	eye := &Eye{}
	eye.active.Store(true)

	if err := eye.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if eye.active.Load() {
		t.Error("Close() should set active to false")
	}
}

func TestEye_Unveil_InvalidConfigType(t *testing.T) {
	eye := &Eye{}
	err := eye.Unveil(context.Background(), testTarget(), wrongConfig{})
	if err == nil {
		t.Fatal("Unveil() should fail with wrong config type")
	}
}

type wrongConfig struct{}

func (wrongConfig) Validate() error { return nil }

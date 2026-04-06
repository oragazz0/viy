package orchestrator

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oragazz0/viy/internal/eyes/disintegration"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/eyes"
)

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

func tempStore(t *testing.T) *state.Store {
	t.Helper()

	directory := t.TempDir()
	return state.NewTestStore(filepath.Join(directory, "state.json"))
}

func newTestOrchestrator(t *testing.T, manager *mockPodManager) *Orchestrator {
	t.Helper()

	logger, _ := zap.NewDevelopment()
	store := tempStore(t)

	return NewOrchestrator(manager, store, logger)
}

func validConfig() *disintegration.Config {
	return &disintegration.Config{
		PodKillCount: 1,
		Strategy:     "sequential",
	}
}

func TestRun_SuccessfulExperiment(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b", "pod-c", "pod-d", "pod-e"),
	}

	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Kind:      "Deployment",
			Name:      "api",
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig:          &disintegration.Config{PodKillCount: 2, Strategy: "sequential"},
		Duration:           1 * time.Minute,
		BlastRadius:        50,
		MinHealthyReplicas: 1,
	}

	err := orch.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(manager.deleted) != 2 {
		t.Errorf("deleted %d pods, want 2", len(manager.deleted))
	}
}

func TestRun_UnknownEye(t *testing.T) {
	manager := &mockPodManager{}
	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName:   "nonexistent",
		EyeConfig: validConfig(),
		Duration:  1 * time.Minute,
	}

	err := orch.Run(context.Background(), config)
	if err == nil {
		t.Fatal("Run() should fail with unknown eye")
	}
}

func TestRun_ValidationFailure(t *testing.T) {
	manager := &mockPodManager{}
	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig: &disintegration.Config{}, // invalid: no kill count or percentage
		Duration:  1 * time.Minute,
	}

	err := orch.Run(context.Background(), config)
	if err == nil {
		t.Fatal("Run() should fail with invalid config")
	}
}

func TestRun_GetPodsError(t *testing.T) {
	manager := &mockPodManager{
		getPodErr: errors.New("connection refused"),
	}

	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig: validConfig(),
		Duration:  1 * time.Minute,
	}

	err := orch.Run(context.Background(), config)
	if err == nil {
		t.Fatal("Run() should propagate GetPods error")
	}
}

func TestRun_BlastRadiusExceeded(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b"),
	}

	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig:          validConfig(),
		Duration:           1 * time.Minute,
		BlastRadius:        50,
		MinHealthyReplicas: 2,
	}

	err := orch.Run(context.Background(), config)
	if err == nil {
		t.Fatal("Run() should fail when blast radius is exceeded")
	}
}

func TestRun_DreamMode(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b", "pod-c"),
	}

	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Kind:      "Deployment",
			Name:      "api",
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig:          validConfig(),
		Duration:           1 * time.Minute,
		BlastRadius:        50,
		MinHealthyReplicas: 1,
		DryRun:             true,
	}

	err := orch.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("Run() dream mode error = %v", err)
	}

	if len(manager.deleted) != 0 {
		t.Error("dream mode should not delete any pods")
	}
}

func TestRun_PersistsExperimentState(t *testing.T) {
	manager := &mockPodManager{
		pods: makePods("pod-a", "pod-b", "pod-c"),
	}

	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Kind:      "Deployment",
			Name:      "api",
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig:          validConfig(),
		Duration:           1 * time.Minute,
		BlastRadius:        50,
		MinHealthyReplicas: 1,
	}

	_ = orch.Run(context.Background(), config)

	experiments, err := orch.store.Load()
	if err != nil {
		t.Fatalf("store.Load() error = %v", err)
	}

	if len(experiments) != 1 {
		t.Fatalf("expected 1 persisted experiment, got %d", len(experiments))
	}

	if experiments[0].Status != state.StatusRevealed {
		t.Errorf("experiment status = %q, want %q", experiments[0].Status, state.StatusRevealed)
	}
}

func TestRun_DeleteError_PersistsFailedState(t *testing.T) {
	manager := &mockPodManager{
		pods:      makePods("pod-a", "pod-b", "pod-c"),
		deleteErr: errors.New("forbidden"),
	}

	orch := newTestOrchestrator(t, manager)

	config := RunConfig{
		EyeName: "disintegration",
		Target: eyes.Target{
			Kind:      "Deployment",
			Name:      "api",
			Namespace: "default",
			Selector:  "app=api",
		},
		EyeConfig:          validConfig(),
		Duration:           1 * time.Minute,
		BlastRadius:        50,
		MinHealthyReplicas: 1,
	}

	_ = orch.Run(context.Background(), config)

	experiments, _ := orch.store.Load()
	if len(experiments) != 1 {
		t.Fatalf("expected 1 persisted experiment, got %d", len(experiments))
	}

	if experiments[0].Status != state.StatusFailed {
		t.Errorf("experiment status = %q, want %q", experiments[0].Status, state.StatusFailed)
	}
}

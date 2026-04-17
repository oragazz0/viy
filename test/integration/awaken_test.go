//go:build integration

// Package integration exercises `viy awaken` end-to-end against a real
// Kubernetes cluster (kind recommended). Build with `-tags=integration`.
//
//	go test -v -race -tags=integration ./test/integration/...
//
// Tests skip cleanly when no cluster is reachable.
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	// Side-effect imports so eye factories + decoders register.
	"github.com/oragazz0/viy/internal/eyes/disintegration"
	_ "github.com/oragazz0/viy/pkg/eyes/death"

	"github.com/oragazz0/viy/internal/k8s"
	"github.com/oragazz0/viy/internal/orchestrator"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/eyes"
	"github.com/oragazz0/viy/pkg/eyes/charm"
)

const (
	testNamespace  = "viy-test"
	testDeployment = "awaken-target"
	expectedReady  = 5
)

func defaultKubeconfig(t *testing.T) string {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no HOME directory: %v", err)
	}

	path := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no kubeconfig at %s: %v", path, err)
	}

	return path
}

func newIntegrationHarness(t *testing.T) (*orchestrator.Orchestrator, *k8s.Client) {
	t.Helper()

	kubeconfig := defaultKubeconfig(t)

	client, err := k8s.NewClient(kubeconfig)
	if err != nil {
		t.Skipf("cannot connect to kubernetes: %v", err)
	}

	if _, err := client.GetPods(context.Background(), testNamespace, ""); err != nil {
		t.Skipf("cluster or namespace %s unreachable: %v", testNamespace, err)
	}

	logger, _ := zap.NewDevelopment()
	store := state.NewTestStore(filepath.Join(t.TempDir(), "state.json"))

	resolver := k8s.NewResolver(client)

	return orchestrator.NewOrchestrator(client, resolver, store, logger), client
}

// TestAwaken_DisintegrationAndCharm exercises both eyes concurrently against
// a real cluster. Prereq:
//
//	kubectl create namespace viy-test
//	kubectl -n viy-test create deployment awaken-target --image=nginx --replicas=5
func TestAwaken_DisintegrationAndCharm(t *testing.T) {
	orch, client := newIntegrationHarness(t)

	disintegrationConfig := &disintegration.Config{
		PodKillCount: 1,
		Strategy:     "random",
		GracePeriod:  5 * time.Second,
	}

	charmConfig := &charm.Config{
		Latency:   200 * time.Millisecond,
		Duration:  30 * time.Second,
		Interface: "eth0",
	}

	multiConfig := orchestrator.MultiConfig{
		ExperimentName: "integration-awaken",
		Duration:       45 * time.Second,
		FailurePolicy:  orchestrator.FailurePolicyContinue,
		BlastRadius:    50,
		MinHealthy:     1,
		Eyes: []orchestrator.EyeRunSpec{
			{
				Name:     "disintegration",
				Target:   targetFor(testDeployment),
				Config:   disintegrationConfig,
				Duration: 30 * time.Second,
			},
			{
				Name:     "charm",
				Target:   targetFor(testDeployment),
				Config:   charmConfig,
				Duration: 30 * time.Second,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := orch.RunMulti(ctx, multiConfig); err != nil {
		t.Fatalf("RunMulti error = %v", err)
	}

	waitForDeploymentRecovery(t, client)
}

func targetFor(deploymentName string) eyes.Target {
	return eyes.Target{
		Kind:      "deployment",
		Name:      deploymentName,
		Namespace: testNamespace,
	}
}

// waitForDeploymentRecovery polls until the deployment has ≥ expectedReady
// Ready pods or the deadline passes.
func waitForDeploymentRecovery(t *testing.T, client *k8s.Client) {
	t.Helper()

	deadline := time.Now().Add(time.Minute)

	for time.Now().Before(deadline) {
		pods, err := client.GetPods(context.Background(), testNamespace, "app="+testDeployment)
		if err == nil && countReady(pods) >= expectedReady {
			return
		}

		time.Sleep(2 * time.Second)
	}

	t.Errorf("deployment did not recover to %d ready pods within 1 minute", expectedReady)
}

func countReady(pods []corev1.Pod) int {
	ready := 0

	for _, pod := range pods {
		if isPodReady(pod) {
			ready++
		}
	}

	return ready
}

func isPodReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

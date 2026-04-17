package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oragazz0/viy/internal/orchestrator"
	"github.com/oragazz0/viy/pkg/config"
	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

const validAwakenYAML = `
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment
metadata:
  name: awaken-test
spec:
  duration: 1m
  failurePolicy: continue
  staggerInterval: 1s
  strictIsolation: false
  safety:
    maxBlastRadius: 30
    minHealthyReplicas: 1
  eyes:
    - name: disintegration
      duration: 30s
      target:
        kind: deployment
        name: api
        namespace: default
      config:
        podKillCount: 1
        interval: 10s
`

func writeExperimentYAML(t *testing.T, body string) string {
	t.Helper()

	directory := t.TempDir()
	path := filepath.Join(directory, "experiment.yaml")

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing YAML: %v", err)
	}

	return path
}

func TestRunAwaken_FileMissing(t *testing.T) {
	err := runAwaken("/nonexistent/does/not/exist.yaml", false)
	if err == nil {
		t.Fatal("runAwaken should fail on missing file")
	}
}

func TestRunAwaken_InvalidYAML(t *testing.T) {
	path := writeExperimentYAML(t, "this: is: :not: valid: :")

	err := runAwaken(path, false)
	if err == nil {
		t.Fatal("runAwaken should fail on invalid YAML")
	}
}

func TestRunAwaken_ValidationFailure(t *testing.T) {
	body := `
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment
metadata:
  name: bad
spec:
  duration: 0s
  eyes: []
`
	path := writeExperimentYAML(t, body)

	err := runAwaken(path, false)
	if err == nil {
		t.Fatal("runAwaken should fail when experiment is invalid")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("error should wrap ErrInvalidConfiguration, got %v", err)
	}
}

func TestRunAwaken_ProtectedNamespace(t *testing.T) {
	body := `
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment
metadata:
  name: protected-ns
spec:
  duration: 1m
  safety:
    maxBlastRadius: 30
  eyes:
    - name: disintegration
      target:
        kind: deployment
        name: api
        namespace: kube-system
      config:
        podKillCount: 1
`
	path := writeExperimentYAML(t, body)

	err := runAwaken(path, false)
	if err == nil {
		t.Fatal("runAwaken should reject protected namespace")
	}
}

func TestBuildMultiConfig_Happy(t *testing.T) {
	path := writeExperimentYAML(t, validAwakenYAML)

	experiment, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}

	if err := experiment.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}

	multiConfig, err := buildMultiConfig(experiment, false)
	if err != nil {
		t.Fatalf("buildMultiConfig error = %v", err)
	}

	if multiConfig.ExperimentName != "awaken-test" {
		t.Errorf("ExperimentName = %q, want awaken-test", multiConfig.ExperimentName)
	}

	if multiConfig.Duration != time.Minute {
		t.Errorf("Duration = %v, want 1m", multiConfig.Duration)
	}

	if multiConfig.FailurePolicy != orchestrator.FailurePolicyContinue {
		t.Errorf("FailurePolicy = %q, want continue", multiConfig.FailurePolicy)
	}

	if multiConfig.StaggerInterval != time.Second {
		t.Errorf("StaggerInterval = %v, want 1s", multiConfig.StaggerInterval)
	}

	if multiConfig.BlastRadius != 30 {
		t.Errorf("BlastRadius = %d, want 30", multiConfig.BlastRadius)
	}

	if len(multiConfig.Eyes) != 1 {
		t.Fatalf("Eyes length = %d, want 1", len(multiConfig.Eyes))
	}

	eye := multiConfig.Eyes[0]
	if eye.Name != "disintegration" {
		t.Errorf("Eye name = %q, want disintegration", eye.Name)
	}

	if eye.Duration != 30*time.Second {
		t.Errorf("Eye duration = %v, want 30s", eye.Duration)
	}

	if eye.Target.Namespace != "default" {
		t.Errorf("Eye target namespace = %q, want default", eye.Target.Namespace)
	}
}

func TestBuildMultiConfig_UnknownEye(t *testing.T) {
	body := `
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment
metadata:
  name: unknown
spec:
  duration: 1m
  safety:
    maxBlastRadius: 30
  eyes:
    - name: nonexistent-eye-decoder
      target:
        kind: deployment
        name: api
        namespace: default
      config: {}
`
	path := writeExperimentYAML(t, body)
	experiment, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}

	if err := experiment.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}

	_, err = buildMultiConfig(experiment, false)
	if err == nil {
		t.Fatal("buildMultiConfig should fail for unknown eye")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("error should wrap ErrInvalidConfiguration, got %v", err)
	}
}

func TestBuildMultiConfig_DryRunFlag(t *testing.T) {
	path := writeExperimentYAML(t, validAwakenYAML)
	experiment, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}

	multiConfig, err := buildMultiConfig(experiment, true)
	if err != nil {
		t.Fatalf("buildMultiConfig error = %v", err)
	}

	if !multiConfig.DryRun {
		t.Error("DryRun should be true when dream flag is set")
	}
}

func TestEnsureAwakenNamespacesAllowed_Protected(t *testing.T) {
	experiment := &config.Experiment{
		Spec: config.Spec{
			Eyes: []config.EyeSpec{
				{
					Name:   "disintegration",
					Target: config.TargetSpec{Namespace: "kube-system"},
				},
			},
		},
	}

	err := ensureAwakenNamespacesAllowed(experiment)
	if err == nil {
		t.Fatal("should reject kube-system namespace")
	}
}

func TestEnsureAwakenNamespacesAllowed_Allowed(t *testing.T) {
	experiment := &config.Experiment{
		Spec: config.Spec{
			Eyes: []config.EyeSpec{
				{
					Name:   "disintegration",
					Target: config.TargetSpec{Namespace: "default"},
				},
				{
					Name:   "charm",
					Target: config.TargetSpec{Namespace: "staging"},
				},
			},
		},
	}

	if err := ensureAwakenNamespacesAllowed(experiment); err != nil {
		t.Errorf("unexpected error for allowed namespaces: %v", err)
	}
}

func TestNewAwakenCommand_RequiresFile(t *testing.T) {
	command := newAwakenCommand()

	if command.Use != "awaken" {
		t.Errorf("command.Use = %q, want awaken", command.Use)
	}

	flag := command.Flags().Lookup("file")
	if flag == nil {
		t.Fatal("awaken command missing --file flag")
	}
}

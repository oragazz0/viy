package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestLoad_ValidYAML(t *testing.T) {
	const body = `
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment
metadata:
  name: demo
spec:
  duration: 5m
  failurePolicy: fail-fast
  staggerInterval: 2s
  strictIsolation: true
  safety:
    maxBlastRadius: 30
    minHealthyReplicas: 1
  eyes:
    - name: disintegration
      duration: 3m
      target:
        kind: deployment
        name: api
        namespace: staging
      config:
        podKillCount: 1
`

	experiment := mustLoadYAML(t, body)

	if experiment.Spec.Duration.ToStd() != 5*time.Minute {
		t.Errorf("Spec.Duration = %v, want 5m", experiment.Spec.Duration.ToStd())
	}

	if experiment.Spec.FailurePolicy != FailurePolicyFailFast {
		t.Errorf("Spec.FailurePolicy = %q, want %q", experiment.Spec.FailurePolicy, FailurePolicyFailFast)
	}

	if !experiment.Spec.StrictIsolation {
		t.Error("Spec.StrictIsolation should be true")
	}

	if len(experiment.Spec.Eyes) != 1 {
		t.Fatalf("Spec.Eyes length = %d, want 1", len(experiment.Spec.Eyes))
	}

	eye := experiment.Spec.Eyes[0]
	if eye.Name != "disintegration" {
		t.Errorf("eye name = %q", eye.Name)
	}

	if eye.Duration.ToStd() != 3*time.Minute {
		t.Errorf("eye.Duration = %v, want 3m", eye.Duration.ToStd())
	}

	killCount, _ := IntField(eye.Config, "podKillCount")
	if killCount != 1 {
		t.Errorf("podKillCount = %d, want 1", killCount)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/to/experiment.yaml")
	if err == nil {
		t.Fatal("Load should fail on missing file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTempYAML(t, "not: valid: yaml: : :")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load should fail on malformed YAML")
	}
}

func TestValidate_Success(t *testing.T) {
	experiment := &Experiment{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   Metadata{Name: "demo"},
		Spec: Spec{
			Duration:      Duration(5 * time.Minute),
			FailurePolicy: FailurePolicyContinue,
			Eyes: []EyeSpec{
				validEyeSpec("disintegration"),
			},
		},
	}

	if err := experiment.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidate_ErrorCases(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(experiment *Experiment)
		wantSubs string
	}{
		{
			name:     "wrong apiVersion",
			mutate:   func(e *Experiment) { e.APIVersion = "foo/v1" },
			wantSubs: "apiVersion",
		},
		{
			name:     "wrong kind",
			mutate:   func(e *Experiment) { e.Kind = "NotAnExperiment" },
			wantSubs: "kind",
		},
		{
			name:     "empty metadata.name",
			mutate:   func(e *Experiment) { e.Metadata.Name = "" },
			wantSubs: "metadata.name",
		},
		{
			name:     "zero duration",
			mutate:   func(e *Experiment) { e.Spec.Duration = 0 },
			wantSubs: "spec.duration",
		},
		{
			name:     "invalid failurePolicy",
			mutate:   func(e *Experiment) { e.Spec.FailurePolicy = "always-fail" },
			wantSubs: "failurePolicy",
		},
		{
			name:     "negative staggerInterval",
			mutate:   func(e *Experiment) { e.Spec.StaggerInterval = Duration(-time.Second) },
			wantSubs: "staggerInterval",
		},
		{
			name:     "no eyes",
			mutate:   func(e *Experiment) { e.Spec.Eyes = nil },
			wantSubs: "spec.eyes",
		},
		{
			name: "duplicate eye name",
			mutate: func(e *Experiment) {
				e.Spec.Eyes = append(e.Spec.Eyes, validEyeSpec("disintegration"))
			},
			wantSubs: "duplicate",
		},
		{
			name: "eye duration exceeds wall-clock",
			mutate: func(e *Experiment) {
				e.Spec.Eyes[0].Duration = Duration(10 * time.Minute)
			},
			wantSubs: "exceeds",
		},
		{
			name: "missing target kind",
			mutate: func(e *Experiment) {
				e.Spec.Eyes[0].Target.Kind = ""
			},
			wantSubs: "target.kind",
		},
		{
			name: "missing target name",
			mutate: func(e *Experiment) {
				e.Spec.Eyes[0].Target.Name = ""
			},
			wantSubs: "target.name",
		},
		{
			name: "missing target namespace",
			mutate: func(e *Experiment) {
				e.Spec.Eyes[0].Target.Namespace = ""
			},
			wantSubs: "target.namespace",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			experiment := validExperiment()
			tc.mutate(experiment)

			err := experiment.Validate()
			if err == nil {
				t.Fatal("Validate() should have failed")
			}

			if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
				t.Errorf("error should wrap ErrInvalidConfiguration, got %v", err)
			}

			if !containsCaseInsensitive(err.Error(), tc.wantSubs) {
				t.Errorf("error message %q should mention %q", err.Error(), tc.wantSubs)
			}
		})
	}
}

func TestFailurePolicy_Resolve(t *testing.T) {
	cases := []struct {
		in   FailurePolicy
		want FailurePolicy
	}{
		{"", FailurePolicyContinue},
		{FailurePolicyContinue, FailurePolicyContinue},
		{FailurePolicyFailFast, FailurePolicyFailFast},
	}

	for _, tc := range cases {
		got := tc.in.Resolve()
		if got != tc.want {
			t.Errorf("FailurePolicy(%q).Resolve() = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func validExperiment() *Experiment {
	return &Experiment{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   Metadata{Name: "demo"},
		Spec: Spec{
			Duration:      Duration(5 * time.Minute),
			FailurePolicy: FailurePolicyContinue,
			Eyes:          []EyeSpec{validEyeSpec("disintegration")},
		},
	}
}

func validEyeSpec(name string) EyeSpec {
	return EyeSpec{
		Name: name,
		Target: TargetSpec{
			Kind:      "deployment",
			Name:      "api",
			Namespace: "staging",
		},
		Config: map[string]any{"podKillCount": 1},
	}
}

func mustLoadYAML(t *testing.T, body string) *Experiment {
	t.Helper()
	path := writeTempYAML(t, body)

	experiment, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	return experiment
}

func writeTempYAML(t *testing.T, body string) string {
	t.Helper()

	directory := t.TempDir()
	path := filepath.Join(directory, "experiment.yaml")

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing temp YAML: %v", err)
	}

	return path
}

func containsCaseInsensitive(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

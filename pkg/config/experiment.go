// Package config parses Viy multi-eye experiment YAML files and dispatches
// heterogeneous per-eye configuration blocks to eye-specific decoders.
//
// The YAML schema is the user-facing contract for `viy awaken`. Each eye
// package registers its own decoder via [RegisterDecoder] so that the
// concrete config type stays owned by the eye, not by this package.
package config

import (
	"fmt"
	"os"
	"time"

	"sigs.k8s.io/yaml"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

// APIVersion is the only apiVersion accepted for ChaosExperiment documents.
const APIVersion = "chaos.viy.io/v1alpha1"

// Kind is the only kind accepted for experiment documents.
const Kind = "ChaosExperiment"

// FailurePolicy controls how sibling eyes react when one eye errors.
type FailurePolicy string

const (
	// FailurePolicyContinue lets siblings keep running when one eye errors.
	// All errors are aggregated at the end. Default.
	FailurePolicyContinue FailurePolicy = "continue"

	// FailurePolicyFailFast cancels sibling eyes via shared context on the
	// first error. Close still runs for every launched eye.
	FailurePolicyFailFast FailurePolicy = "fail-fast"
)

// Experiment is the root YAML document for `viy awaken`.
type Experiment struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
}

// Metadata carries identifying information for an experiment.
type Metadata struct {
	Name string `json:"name"`
}

// Spec holds the experiment-level configuration shared across eyes.
type Spec struct {
	Duration        Duration      `json:"duration"`
	FailurePolicy   FailurePolicy `json:"failurePolicy,omitempty"`
	StaggerInterval Duration      `json:"staggerInterval,omitempty"`
	StrictIsolation bool          `json:"strictIsolation,omitempty"`
	Safety          SafetySpec    `json:"safety"`
	Eyes            []EyeSpec     `json:"eyes"`
}

// SafetySpec holds blast radius limits applied per eye.
type SafetySpec struct {
	MaxBlastRadius     int `json:"maxBlastRadius"`
	MinHealthyReplicas int `json:"minHealthyReplicas"`
}

// EyeSpec describes one eye instance within a multi-eye experiment.
// Config is opaque YAML; [DecodeConfig] resolves it to the typed
// config owned by the eye package.
type EyeSpec struct {
	Name     string         `json:"name"`
	Duration Duration       `json:"duration,omitempty"`
	Target   TargetSpec     `json:"target"`
	Config   map[string]any `json:"config"`
}

// TargetSpec identifies a Kubernetes resource this eye acts on.
type TargetSpec struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Selector  string `json:"selector,omitempty"`
}

// Load reads and parses an experiment YAML file from disk.
func Load(path string) (*Experiment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading experiment file %q: %w", path, err)
	}

	var exp Experiment
	if err := yaml.Unmarshal(data, &exp); err != nil {
		return nil, fmt.Errorf("parsing experiment YAML: %w", err)
	}

	return &exp, nil
}

// Validate performs structural checks on the experiment. It does not
// validate per-eye config — that happens after [DecodeConfig] produces
// the typed config.
func (e *Experiment) Validate() error {
	if e.APIVersion != APIVersion {
		return fmt.Errorf("%w: apiVersion must be %q, got %q",
			viyerrors.ErrInvalidConfiguration, APIVersion, e.APIVersion)
	}

	if e.Kind != Kind {
		return fmt.Errorf("%w: kind must be %q, got %q",
			viyerrors.ErrInvalidConfiguration, Kind, e.Kind)
	}

	if e.Metadata.Name == "" {
		return fmt.Errorf("%w: metadata.name is required",
			viyerrors.ErrInvalidConfiguration)
	}

	return e.Spec.validate()
}

func (s *Spec) validate() error {
	if s.Duration.ToStd() <= 0 {
		return fmt.Errorf("%w: spec.duration must be positive",
			viyerrors.ErrInvalidConfiguration)
	}

	if err := s.FailurePolicy.validate(); err != nil {
		return err
	}

	if s.StaggerInterval.ToStd() < 0 {
		return fmt.Errorf("%w: spec.staggerInterval must be non-negative",
			viyerrors.ErrInvalidConfiguration)
	}

	if len(s.Eyes) == 0 {
		return fmt.Errorf("%w: spec.eyes must contain at least one eye",
			viyerrors.ErrInvalidConfiguration)
	}

	return s.validateEyes()
}

func (s *Spec) validateEyes() error {
	seen := make(map[string]bool, len(s.Eyes))

	for index, eye := range s.Eyes {
		if err := eye.validate(s.Duration.ToStd()); err != nil {
			return fmt.Errorf("eyes[%d] (%s): %w", index, eye.Name, err)
		}

		if seen[eye.Name] {
			return fmt.Errorf("%w: duplicate eye name %q",
				viyerrors.ErrInvalidConfiguration, eye.Name)
		}

		seen[eye.Name] = true
	}

	return nil
}

func (p FailurePolicy) validate() error {
	switch p {
	case "", FailurePolicyContinue, FailurePolicyFailFast:
		return nil
	default:
		return fmt.Errorf("%w: failurePolicy must be %q or %q, got %q",
			viyerrors.ErrInvalidConfiguration,
			FailurePolicyContinue, FailurePolicyFailFast, p)
	}
}

// Resolve returns the failure policy with the default applied when unset.
func (p FailurePolicy) Resolve() FailurePolicy {
	if p == "" {
		return FailurePolicyContinue
	}

	return p
}

func (e *EyeSpec) validate(wallClock time.Duration) error {
	if e.Name == "" {
		return fmt.Errorf("%w: eye name is required",
			viyerrors.ErrInvalidConfiguration)
	}

	if e.Duration.ToStd() < 0 {
		return fmt.Errorf("%w: eye duration must be non-negative",
			viyerrors.ErrInvalidConfiguration)
	}

	if e.Duration.ToStd() > wallClock {
		return fmt.Errorf("%w: eye duration %s exceeds spec.duration %s",
			viyerrors.ErrInvalidConfiguration, e.Duration.ToStd(), wallClock)
	}

	return e.Target.validate()
}

func (t *TargetSpec) validate() error {
	if t.Kind == "" {
		return fmt.Errorf("%w: target.kind is required",
			viyerrors.ErrInvalidConfiguration)
	}

	if t.Name == "" {
		return fmt.Errorf("%w: target.name is required",
			viyerrors.ErrInvalidConfiguration)
	}

	if t.Namespace == "" {
		return fmt.Errorf("%w: target.namespace is required",
			viyerrors.ErrInvalidConfiguration)
	}

	return nil
}

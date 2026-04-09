package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/oragazz0/viy/internal/k8s"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/eyes"
	"github.com/oragazz0/viy/pkg/safety"
)

// Orchestrator wires target resolution, safety checks, and eye execution.
type Orchestrator struct {
	podManager k8s.PodManager
	resolver   k8s.TargetResolver
	store      *state.Store
	logger     *zap.Logger
}

// NewOrchestrator creates an Orchestrator with all dependencies.
func NewOrchestrator(podManager k8s.PodManager, resolver k8s.TargetResolver, store *state.Store, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		podManager: podManager,
		resolver:   resolver,
		store:      store,
		logger:     logger,
	}
}

// RunConfig carries everything needed to start an experiment.
type RunConfig struct {
	EyeName            string
	Target             eyes.Target
	EyeConfig          eyes.EyeConfig
	Duration           time.Duration
	BlastRadius        int
	MinHealthyReplicas int
	DryRun             bool
}

// Run executes a single-eye experiment end-to-end.
func (o *Orchestrator) Run(ctx context.Context, config RunConfig) error {
	experimentID := uuid.New().String()[:12]
	o.logger.Info("experiment starting",
		zap.String("experiment_id", experimentID),
		zap.String("eye", config.EyeName),
		zap.String("target", config.Target.Name),
		zap.String("namespace", config.Target.Namespace),
		zap.Bool("dry_run", config.DryRun),
	)

	eye, err := eyes.Get(config.EyeName, eyes.Dependencies{
		PodManager: o.podManager,
		Logger:     o.logger,
	})
	if err != nil {
		return err
	}

	if err := eye.Validate(config.EyeConfig); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	resolved, err := o.resolver.Resolve(ctx, config.Target)
	if err != nil {
		return fmt.Errorf("target resolution: %w", err)
	}

	maxAffected, err := safety.CalculateMaxAffected(len(resolved.Pods), safety.BlastRadiusConfig{
		MaxPercentage:      config.BlastRadius,
		MinHealthyReplicas: config.MinHealthyReplicas,
	})
	if err != nil {
		return err
	}

	o.logger.Info("targets resolved",
		zap.String("resource", resolved.ResourceKind+"/"+resolved.ResourceName),
		zap.String("selector", resolved.Selector),
		zap.Int("total_pods", len(resolved.Pods)),
		zap.Int("max_affected", maxAffected),
		zap.Int("blast_radius_pct", config.BlastRadius),
	)

	if config.DryRun {
		return o.runDreamMode(resolved, config, maxAffected)
	}

	experiment := state.Experiment{
		ID:        experimentID,
		Status:    state.StatusUnveiling,
		Eyes:      []string{config.EyeName},
		Target:    config.Target.Name,
		Namespace: config.Target.Namespace,
		StartTime: time.Now(),
		Duration:  config.Duration,
	}

	if err := o.saveExperiment(experiment); err != nil {
		o.logger.Warn("failed to persist experiment state", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	err = eye.Unveil(ctx, config.Target, config.EyeConfig)

	now := time.Now()
	experiment.EndTime = &now
	if err != nil {
		experiment.Status = state.StatusFailed
	} else {
		experiment.Status = state.StatusRevealed
	}

	if saveErr := o.saveExperiment(experiment); saveErr != nil {
		o.logger.Warn("failed to persist experiment state", zap.Error(saveErr))
	}

	if err != nil {
		return fmt.Errorf("revelation failed: %w", err)
	}

	o.logger.Info("revelation complete",
		zap.String("experiment_id", experimentID),
	)

	return nil
}

func (o *Orchestrator) runDreamMode(resolved *k8s.ResolvedTarget, config RunConfig, maxAffected int) error {
	fmt.Println()
	fmt.Println("🔮 Dream Mode: Viy dreams of revelation...")
	fmt.Println()

	fmt.Println("Target resolution:")
	fmt.Printf("  Resource: %s/%s (%s) — found ✓\n",
		resolved.ResourceKind, resolved.ResourceName, config.Target.Namespace)
	fmt.Printf("  Selector: %s\n", resolved.Selector)
	fmt.Printf("  Pods matched: %d\n", len(resolved.Pods))
	fmt.Println()

	limit := maxAffected
	if limit > len(resolved.Pods) {
		limit = len(resolved.Pods)
	}

	fmt.Println("Targets that would be unveiled:")

	for _, pod := range resolved.Pods[:limit] {
		fmt.Printf("  • Pod: %s (%s)\n", pod.Name, pod.Namespace)
	}

	fmt.Println()
	fmt.Printf("Estimated blast radius: %d%% (%d/%d pods)\n",
		config.BlastRadius, limit, len(resolved.Pods))
	fmt.Println("Safety checks: ✅ All passed")

	return nil
}

func (o *Orchestrator) saveExperiment(experiment state.Experiment) error {
	experiments, err := o.store.Load()
	if err != nil {
		return err
	}

	found := false
	for index := range experiments {
		if experiments[index].ID == experiment.ID {
			experiments[index] = experiment
			found = true
			break
		}
	}

	if !found {
		experiments = append(experiments, experiment)
	}

	return o.store.Save(experiments)
}

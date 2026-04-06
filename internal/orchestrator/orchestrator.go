package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/oragazz0/viy/internal/k8s"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/eyes"
	"github.com/oragazz0/viy/pkg/safety"
)

// Orchestrator wires target resolution, safety checks, and eye execution.
type Orchestrator struct {
	podManager k8s.PodManager
	store      *state.Store
	logger     *zap.Logger
}

// NewOrchestrator creates an Orchestrator with all dependencies.
func NewOrchestrator(podManager k8s.PodManager, store *state.Store, logger *zap.Logger) *Orchestrator {
	return &Orchestrator{
		podManager: podManager,
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

	eye, err := eyes.Get(config.EyeName)
	if err != nil {
		return err
	}

	eye.Init(o.podManager, o.logger)

	if err := eye.Validate(config.EyeConfig); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	pods, err := o.podManager.GetPods(ctx, config.Target.Namespace, config.Target.Selector)
	if err != nil {
		return fmt.Errorf("target resolution: %w", err)
	}

	maxAffected, err := safety.CalculateMaxAffected(len(pods), safety.BlastRadiusConfig{
		MaxPercentage:      config.BlastRadius,
		MinHealthyReplicas: config.MinHealthyReplicas,
	})
	if err != nil {
		return err
	}

	o.logger.Info("targets resolved",
		zap.Int("total_pods", len(pods)),
		zap.Int("max_affected", maxAffected),
		zap.Int("blast_radius_pct", config.BlastRadius),
	)

	if config.DryRun {
		return o.runDreamMode(pods, config, maxAffected)
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

func (o *Orchestrator) runDreamMode(pods []corev1.Pod, config RunConfig, maxAffected int) error {
	fmt.Println()
	fmt.Println("🔮 Dream Mode: Viy dreams of revelation...")
	fmt.Println()
	fmt.Printf("Targets that would be unveiled:\n")

	limit := maxAffected
	if limit > len(pods) {
		limit = len(pods)
	}

	for _, pod := range pods[:limit] {
		fmt.Printf("  • Pod: %s (%s/%s)\n", pod.Name, pod.Namespace, config.Target.Name)
	}

	fmt.Println()
	fmt.Printf("Estimated blast radius: %d%% (%d/%d pods)\n",
		config.BlastRadius, limit, len(pods))
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

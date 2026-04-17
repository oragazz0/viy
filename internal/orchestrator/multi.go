package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/oragazz0/viy/internal/k8s"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/eyes"
)

// FailurePolicy controls how sibling eyes react when one eye errors.
type FailurePolicy string

const (
	// FailurePolicyContinue lets siblings run; errors aggregated at the end.
	FailurePolicyContinue FailurePolicy = "continue"

	// FailurePolicyFailFast cancels siblings via shared context on first error.
	FailurePolicyFailFast FailurePolicy = "fail-fast"
)

// closeTimeout bounds graceful cleanup once an eye's Unveil has returned
// (or been cancelled). Uses a fresh background context so cleanup survives
// sibling cancellation — critical for eyes that call ExecInContainer in Close.
const closeTimeout = 30 * time.Second

// aggregationInterval drives the periodic Observe-snapshot log.
const aggregationInterval = 10 * time.Second

// MultiConfig carries everything needed to run a multi-eye experiment.
type MultiConfig struct {
	ExperimentName  string
	Duration        time.Duration
	FailurePolicy   FailurePolicy
	StaggerInterval time.Duration
	StrictIsolation bool
	BlastRadius     int
	MinHealthy      int
	DryRun          bool
	Eyes            []EyeRunSpec
}

// EyeRunSpec is one eye's participation in a multi-eye experiment.
// Duration == 0 inherits MultiConfig.Duration.
type EyeRunSpec struct {
	Name     string
	Target   eyes.Target
	Config   eyes.EyeConfig
	Duration time.Duration
}

// eyeHandle is the fully prepared state for one eye, produced during
// RunMulti setup and consumed by the launch loop.
type eyeHandle struct {
	name     string
	eye      eyes.Eye
	target   eyes.Target
	config   eyes.EyeConfig
	duration time.Duration
	resolved *k8s.ResolvedTarget
}

// RunMulti executes a multi-eye experiment concurrently.
//
// Lifecycle: build+validate every eye → resolve every target → per-eye
// blast radius check → contention detection → launch under shared ctx
// with chosen failure policy → aggregate metrics → wait → Close every
// launched eye with a fresh context.
func (o *Orchestrator) RunMulti(ctx context.Context, cfg MultiConfig) error {
	experimentID := uuid.New().String()[:12]

	o.logger.Info("Viy's gaze focuses on multiple eyes",
		zap.String("experiment_id", experimentID),
		zap.String("name", cfg.ExperimentName),
		zap.Int("eye_count", len(cfg.Eyes)),
		zap.String("failure_policy", string(cfg.FailurePolicy)),
		zap.Duration("duration", cfg.Duration),
		zap.Bool("dry_run", cfg.DryRun),
	)

	if len(cfg.Eyes) == 0 {
		return fmt.Errorf("no eyes to awaken")
	}

	handles, err := o.prepareHandles(ctx, cfg)
	if err != nil {
		return err
	}

	if err := o.enforceContention(handles, cfg.StrictIsolation); err != nil {
		return err
	}

	if cfg.DryRun {
		return o.runMultiDreamMode(handles, cfg)
	}

	experiment := o.newMultiExperiment(experimentID, cfg)
	if err := o.saveExperiment(experiment); err != nil {
		o.logger.Warn("failed to persist experiment state", zap.Error(err))
	}

	runErr := o.launchAll(ctx, handles, cfg)

	o.finalizeExperiment(&experiment, runErr)
	if err := o.saveExperiment(experiment); err != nil {
		o.logger.Warn("failed to persist experiment state", zap.Error(err))
	}

	if runErr != nil {
		return fmt.Errorf("revelation failed: %w", runErr)
	}

	o.logger.Info("all eyes have closed — revelations complete",
		zap.String("experiment_id", experimentID),
	)

	return nil
}

// prepareHandles builds each eye, validates its config, resolves its
// target, and checks per-eye blast radius. Returns on the first failure.
func (o *Orchestrator) prepareHandles(ctx context.Context, cfg MultiConfig) ([]eyeHandle, error) {
	handles := make([]eyeHandle, 0, len(cfg.Eyes))

	for _, spec := range cfg.Eyes {
		handle, err := o.prepareOne(ctx, spec, cfg)
		if err != nil {
			return nil, fmt.Errorf("preparing eye %q: %w", spec.Name, err)
		}

		handles = append(handles, handle)
	}

	return handles, nil
}

func (o *Orchestrator) prepareOne(ctx context.Context, spec EyeRunSpec, cfg MultiConfig) (eyeHandle, error) {
	eye, err := o.buildEye(spec.Name)
	if err != nil {
		return eyeHandle{}, err
	}

	if err := eye.Validate(spec.Config); err != nil {
		return eyeHandle{}, fmt.Errorf("validation failed: %w", err)
	}

	resolved, err := o.resolveTarget(ctx, spec.Target)
	if err != nil {
		return eyeHandle{}, err
	}

	maxAffected, err := o.checkBlastRadius(len(resolved.Pods), cfg.BlastRadius, cfg.MinHealthy)
	if err != nil {
		return eyeHandle{}, err
	}

	o.logger.Info("eye prepared",
		zap.String("eye", spec.Name),
		zap.String("resource", resolved.ResourceKind+"/"+resolved.ResourceName),
		zap.Int("total_pods", len(resolved.Pods)),
		zap.Int("max_affected", maxAffected),
	)

	return eyeHandle{
		name:     spec.Name,
		eye:      eye,
		target:   spec.Target,
		config:   spec.Config,
		duration: resolveDuration(spec.Duration, cfg.Duration),
		resolved: resolved,
	}, nil
}

func resolveDuration(eye, wallClock time.Duration) time.Duration {
	if eye > 0 {
		return eye
	}

	return wallClock
}

// enforceContention warns (or rejects on strictIsolation) when two eyes
// target the same pod. Runs before any Unveil.
func (o *Orchestrator) enforceContention(handles []eyeHandle, strict bool) error {
	resolutions := make(map[string]*k8s.ResolvedTarget, len(handles))
	for _, handle := range handles {
		resolutions[handle.name] = handle.resolved
	}

	overlaps := detectOverlap(resolutions)
	if len(overlaps) == 0 {
		return nil
	}

	if strict {
		return fmt.Errorf("strict isolation: %w", newContentionError(overlaps))
	}

	for _, overlap := range overlaps {
		o.logger.Warn("eyes gaze upon the same pod",
			zap.String("pod", overlap.PodName),
			zap.String("namespace", overlap.Namespace),
			zap.Strings("eyes", overlap.Eyes),
		)
	}

	return nil
}

// launchAll runs every eye concurrently under the chosen failure policy,
// bounded by the wall-clock duration, with metrics aggregation and
// guaranteed Close for every launched eye.
func (o *Orchestrator) launchAll(ctx context.Context, handles []eyeHandle, cfg MultiConfig) error {
	runCtx, cancelRun := context.WithTimeout(ctx, cfg.Duration)
	defer cancelRun()

	aggregatorCtx, stopAggregator := context.WithCancel(runCtx)
	defer stopAggregator()

	go o.aggregateMetrics(aggregatorCtx, handles)

	policy := cfg.FailurePolicy
	if policy == "" {
		policy = FailurePolicyContinue
	}

	if policy == FailurePolicyFailFast {
		return o.launchFailFast(runCtx, handles, cfg.StaggerInterval)
	}

	return o.launchContinue(runCtx, handles, cfg.StaggerInterval)
}

func (o *Orchestrator) launchFailFast(runCtx context.Context, handles []eyeHandle, stagger time.Duration) error {
	group, groupCtx := errgroup.WithContext(runCtx)

	for index, handle := range handles {
		if err := waitForStagger(groupCtx, stagger, index); err != nil {
			return err
		}

		handleCopy := handle
		group.Go(func() error {
			return o.runOne(groupCtx, handleCopy)
		})
	}

	return group.Wait()
}

func (o *Orchestrator) launchContinue(runCtx context.Context, handles []eyeHandle, stagger time.Duration) error {
	var waitGroup sync.WaitGroup
	errs := make([]error, len(handles))

	for index, handle := range handles {
		if err := waitForStagger(runCtx, stagger, index); err != nil {
			return err
		}

		waitGroup.Add(1)
		handleCopy := handle
		slot := index
		go func() {
			defer waitGroup.Done()
			errs[slot] = o.runOne(runCtx, handleCopy)
		}()
	}

	waitGroup.Wait()

	return errors.Join(errs...)
}

// runOne executes one eye's Unveil under a per-eye timeout and guarantees
// Close runs with a fresh context, even on panic or parent cancellation.
func (o *Orchestrator) runOne(groupCtx context.Context, handle eyeHandle) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("eye %s panicked: %v", handle.name, recovered)
			o.logger.Error("eye panicked",
				zap.String("eye", handle.name),
				zap.Any("panic", recovered),
			)
		}

		closeCtx, cancelClose := context.WithTimeout(context.Background(), closeTimeout)
		defer cancelClose()

		if closeErr := handle.eye.Close(closeCtx); closeErr != nil {
			o.logger.Warn("close failed",
				zap.String("eye", handle.name),
				zap.Error(closeErr),
			)
		}
	}()

	runCtx, cancel := context.WithTimeout(groupCtx, handle.duration)
	defer cancel()

	o.logger.Info("opening eye",
		zap.String("eye", handle.name),
		zap.Duration("duration", handle.duration),
	)

	unveilErr := handle.eye.Unveil(runCtx, handle.target, handle.config)
	if unveilErr != nil {
		return fmt.Errorf("eye %s: %w", handle.name, unveilErr)
	}

	return nil
}

// waitForStagger sleeps between launches so eyes don't all fire at t=0.
// index 0 never waits.
func waitForStagger(ctx context.Context, stagger time.Duration, index int) error {
	if stagger <= 0 || index == 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(stagger):
		return nil
	}
}

// aggregateMetrics logs a per-eye Observe snapshot every
// aggregationInterval until the context is cancelled.
func (o *Orchestrator) aggregateMetrics(ctx context.Context, handles []eyeHandle) {
	ticker := time.NewTicker(aggregationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.logAggregatedMetrics(handles)
		}
	}
}

func (o *Orchestrator) logAggregatedMetrics(handles []eyeHandle) {
	for _, handle := range handles {
		metrics := handle.eye.Observe()
		o.logger.Info("eye metrics",
			zap.String("eye", metrics.EyeName),
			zap.Int("targets_affected", metrics.TargetsAffected),
			zap.Int64("operations_total", metrics.OperationsTotal),
			zap.Int64("errors_total", metrics.ErrorsTotal),
			zap.Bool("active", metrics.IsActive),
		)
	}
}

func (o *Orchestrator) newMultiExperiment(experimentID string, cfg MultiConfig) state.Experiment {
	eyeNames := make([]string, 0, len(cfg.Eyes))
	for _, spec := range cfg.Eyes {
		eyeNames = append(eyeNames, spec.Name)
	}

	targetSummary, namespaceSummary := summarizeTargets(cfg.Eyes)

	return state.Experiment{
		ID:        experimentID,
		Status:    state.StatusUnveiling,
		Eyes:      eyeNames,
		Target:    targetSummary,
		Namespace: namespaceSummary,
		StartTime: time.Now(),
		Duration:  cfg.Duration,
	}
}

func summarizeTargets(specs []EyeRunSpec) (target, namespace string) {
	if len(specs) == 0 {
		return "", ""
	}

	target = specs[0].Target.Name
	namespace = specs[0].Target.Namespace

	for _, spec := range specs[1:] {
		if spec.Target.Name != target {
			target = "multiple"
		}

		if spec.Target.Namespace != namespace {
			namespace = "multiple"
		}
	}

	return target, namespace
}

func (o *Orchestrator) finalizeExperiment(experiment *state.Experiment, runErr error) {
	now := time.Now()
	experiment.EndTime = &now

	if runErr != nil {
		experiment.Status = state.StatusFailed
		return
	}

	experiment.Status = state.StatusRevealed
}

func (o *Orchestrator) runMultiDreamMode(handles []eyeHandle, cfg MultiConfig) error {
	fmt.Println()
	fmt.Println("🔮 Dream Mode: Viy dreams of collective revelation...")
	fmt.Println()
	fmt.Printf("Experiment: %s\n", cfg.ExperimentName)
	fmt.Printf("Wall-clock duration: %s\n", cfg.Duration)
	fmt.Printf("Failure policy: %s\n", cfg.FailurePolicy)
	fmt.Printf("Stagger interval: %s\n", cfg.StaggerInterval)
	fmt.Printf("Strict isolation: %t\n", cfg.StrictIsolation)
	fmt.Println()

	for _, handle := range handles {
		fmt.Printf("Eye: %s (duration %s)\n", handle.name, handle.duration)
		fmt.Printf("  Target: %s/%s (%s)\n",
			handle.resolved.ResourceKind, handle.resolved.ResourceName, handle.target.Namespace)
		fmt.Printf("  Selector: %s\n", handle.resolved.Selector)
		fmt.Printf("  Pods matched: %d\n", len(handle.resolved.Pods))
	}

	fmt.Println()
	fmt.Println("Safety checks: ✅ All passed")

	return nil
}

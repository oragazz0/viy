// Package death implements the Eye of Death — a chaos module that reveals
// truths about resource limits, HPA scaling, and OOM killer behavior through
// controlled resource exhaustion via stress-ng ephemeral container injection.
package death

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

const (
	eyeName         = "death"
	stressImage     = "alexeiled/stress-ng:latest"
	containerPrefix = "viy-death-"
)

func init() {
	eyes.Register(eyeName, NewDeathEye)
}

// NewDeathEye creates a new Eye of Death with the given dependencies.
func NewDeathEye(deps eyes.Dependencies) eyes.Eye {
	return &DeathEye{
		podManager:          deps.PodManager,
		ephemeralContainers: deps.EphemeralContainerManager,
		logger:              deps.Logger,
	}
}

type injectedStress struct {
	namespace     string
	podName       string
	containerName string
}

// DeathEye reveals truths about resource limits through controlled exhaustion.
type DeathEye struct {
	podManager          eyes.PodManager
	ephemeralContainers eyes.EphemeralContainerManager
	logger              *zap.Logger

	mu                sync.Mutex
	injected          []injectedStress
	truthsRevealed    []string
	active            atomic.Bool
	targetsAffected   atomic.Int64
	operationsTotal   atomic.Int64
	errorsTotal       atomic.Int64
	lastExecutionTime atomic.Value
}

func (e *DeathEye) Name() string {
	return eyeName
}

func (e *DeathEye) Description() string {
	return "Reveals truths about resource limits, HPA scaling, and OOM killer behavior through controlled resource exhaustion"
}

func (e *DeathEye) Validate(config eyes.EyeConfig) error {
	if _, ok := config.(*Config); !ok {
		return fmt.Errorf("%w: expected *death.Config", viyerrors.ErrInvalidConfiguration)
	}

	return config.Validate()
}

func (e *DeathEye) Unveil(ctx context.Context, target eyes.Target, config eyes.EyeConfig) error {
	cfg := config.(*Config)

	e.active.Store(true)
	e.logger.Info("opening Eye of Death",
		zap.String("target", target.Name),
		zap.String("namespace", target.Namespace),
	)

	pods, err := e.podManager.GetPods(ctx, target.Namespace, target.Selector)
	if err != nil {
		e.errorsTotal.Add(1)
		return fmt.Errorf("failed to discover targets for death's gaze: %w", err)
	}

	if len(pods) == 0 {
		return fmt.Errorf(
			"%w: no pods match selector %q in namespace %q",
			viyerrors.ErrTargetNotFound, target.Selector, target.Namespace,
		)
	}

	command := buildStressCommand(cfg)
	injectedCount, failedCount := e.injectStressIntoPods(ctx, pods, target.Namespace, command)

	e.lastExecutionTime.Store(time.Now())
	e.addTruth(fmt.Sprintf("death's grasp reached %d pods in namespace %q", injectedCount, target.Namespace))

	if failedCount > 0 {
		e.addTruth(fmt.Sprintf("death could not reach %d pods — partial revelation", failedCount))
	}

	return nil
}

func (e *DeathEye) injectStressIntoPods(ctx context.Context, pods []corev1.Pod, namespace string, command []string) (injected, failed int) {
	for _, pod := range pods {
		if ctx.Err() != nil {
			return injected, failed
		}

		containerName := containerPrefix + truncatePodName(pod.Name)
		ephContainer := buildEphemeralContainer(containerName, command)

		err := e.ephemeralContainers.AddEphemeralContainer(ctx, namespace, pod.Name, ephContainer)
		if err != nil {
			e.errorsTotal.Add(1)
			e.logger.Error("failed to inject death into pod",
				zap.String("pod", pod.Name),
				zap.Error(err),
			)
			failed++
			continue
		}

		e.trackInjection(namespace, pod.Name, containerName)
		e.targetsAffected.Add(1)
		e.operationsTotal.Add(1)
		e.logger.Info("death's grasp tightens", zap.String("pod", pod.Name))
		injected++
	}

	return injected, failed
}

func (e *DeathEye) Pause(ctx context.Context) error {
	return e.terminateAllStress(ctx)
}

func (e *DeathEye) Close(ctx context.Context) error {
	defer e.active.Store(false)
	return e.terminateAllStress(ctx)
}

func (e *DeathEye) Observe() eyes.Metrics {
	var lastExecution time.Time
	if stored := e.lastExecutionTime.Load(); stored != nil {
		lastExecution = stored.(time.Time)
	}

	e.mu.Lock()
	truths := make([]string, len(e.truthsRevealed))
	copy(truths, e.truthsRevealed)
	e.mu.Unlock()

	return eyes.Metrics{
		EyeName:           eyeName,
		TargetsAffected:   int(e.targetsAffected.Load()),
		OperationsTotal:   e.operationsTotal.Load(),
		ErrorsTotal:       e.errorsTotal.Load(),
		TruthsRevealed:    truths,
		LastExecutionTime: lastExecution,
		IsActive:          e.active.Load(),
	}
}

func (e *DeathEye) terminateAllStress(ctx context.Context) error {
	e.mu.Lock()
	containers := make([]injectedStress, len(e.injected))
	copy(containers, e.injected)
	e.mu.Unlock()

	for _, stress := range containers {
		killCommand := []string{"kill", "1"}
		err := e.ephemeralContainers.ExecInContainer(ctx, stress.namespace, stress.podName, stress.containerName, killCommand)
		if err != nil {
			e.logger.Warn("failed to terminate stress in pod",
				zap.String("pod", stress.podName),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (e *DeathEye) trackInjection(namespace, podName, containerName string) {
	e.mu.Lock()
	e.injected = append(e.injected, injectedStress{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
	})
	e.mu.Unlock()
}

func (e *DeathEye) addTruth(truth string) {
	e.mu.Lock()
	e.truthsRevealed = append(e.truthsRevealed, truth)
	e.mu.Unlock()
}

func buildStressCommand(cfg *Config) []string {
	args := []string{"stress-ng"}

	if cfg.CPUStressPercent > 0 {
		args = append(args,
			"--cpu", fmt.Sprintf("%d", cfg.Workers),
			"--cpu-load", fmt.Sprintf("%d", cfg.CPUStressPercent),
		)
	}

	if cfg.MemoryStressPercent > 0 {
		args = append(args,
			"--vm", fmt.Sprintf("%d", cfg.Workers),
			"--vm-bytes", fmt.Sprintf("%d%%", cfg.MemoryStressPercent),
		)
	}

	if cfg.DiskIOBytes > 0 {
		args = append(args,
			"--hdd", fmt.Sprintf("%d", cfg.Workers),
			"--hdd-bytes", fmt.Sprintf("%d", cfg.DiskIOBytes),
		)
	}

	if cfg.RampUp > 0 {
		args = append(args, "--ramp-time", fmt.Sprintf("%d", int(cfg.RampUp.Seconds())))
	}

	args = append(args, "--timeout", fmt.Sprintf("%d", int(cfg.Duration.Seconds())))

	return args
}

func buildEphemeralContainer(name string, command []string) corev1.EphemeralContainer {
	return corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:    name,
			Image:   stressImage,
			Command: command,
		},
	}
}

func truncatePodName(name string) string {
	const maxLength = 8
	if len(name) <= maxLength {
		return name
	}

	return name[:maxLength]
}

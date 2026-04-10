// Package charm implements the Eye of Charm — a chaos module that reveals
// truths about network dependencies, timeout configurations, and circuit
// breaker behavior through controlled network manipulation via tc netem.
package charm

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
	eyeName         = "charm"
	netshootImage   = "nicolaka/netshoot:latest"
	containerPrefix = "viy-charm-"
)

func init() {
	eyes.Register(eyeName, NewCharmEye)
}

// NewCharmEye creates a new Eye of Charm with the given dependencies.
func NewCharmEye(deps eyes.Dependencies) eyes.Eye {
	return &CharmEye{
		podManager:          deps.PodManager,
		ephemeralContainers: deps.EphemeralContainerManager,
		logger:              deps.Logger,
	}
}

type charmedPod struct {
	namespace     string
	podName       string
	containerName string
	iface         string
}

// CharmEye reveals truths about network resilience through controlled chaos.
type CharmEye struct {
	podManager          eyes.PodManager
	ephemeralContainers eyes.EphemeralContainerManager
	logger              *zap.Logger

	mu                sync.Mutex
	charmed           []charmedPod
	truthsRevealed    []string
	active            atomic.Bool
	targetsAffected   atomic.Int64
	operationsTotal   atomic.Int64
	errorsTotal       atomic.Int64
	lastExecutionTime atomic.Value
}

func (e *CharmEye) Name() string {
	return eyeName
}

func (e *CharmEye) Description() string {
	return "Reveals truths about network dependencies, timeouts, and circuit breaker behavior through controlled network manipulation"
}

func (e *CharmEye) Validate(config eyes.EyeConfig) error {
	if _, ok := config.(*Config); !ok {
		return fmt.Errorf("%w: expected *charm.Config", viyerrors.ErrInvalidConfiguration)
	}

	return config.Validate()
}

func (e *CharmEye) Unveil(ctx context.Context, target eyes.Target, config eyes.EyeConfig) error {
	cfg := config.(*Config)

	e.active.Store(true)
	e.logger.Info("opening Eye of Charm",
		zap.String("target", target.Name),
		zap.String("namespace", target.Namespace),
	)

	pods, err := e.podManager.GetPods(ctx, target.Namespace, target.Selector)
	if err != nil {
		e.errorsTotal.Add(1)
		return fmt.Errorf("failed to reveal network truths: %w", err)
	}

	if len(pods) == 0 {
		return fmt.Errorf(
			"%w: no pods match selector %q in namespace %q",
			viyerrors.ErrTargetNotFound, target.Selector, target.Namespace,
		)
	}

	charmedCount, failedCount := e.applyCharmToPods(ctx, pods, target.Namespace, cfg)

	e.lastExecutionTime.Store(time.Now())
	e.addTruth(fmt.Sprintf("charm woven across %d pods in namespace %q", charmedCount, target.Namespace))

	if failedCount > 0 {
		e.addTruth(fmt.Sprintf("charm could not reach %d pods — partial enchantment", failedCount))
	}

	return nil
}

func (e *CharmEye) applyCharmToPods(ctx context.Context, pods []corev1.Pod, namespace string, cfg *Config) (charmed, failed int) {
	for _, pod := range pods {
		if ctx.Err() != nil {
			return charmed, failed
		}

		containerName := containerPrefix + truncatePodName(pod.Name)

		if err := e.charmPod(ctx, namespace, pod.Name, containerName, cfg); err != nil {
			e.errorsTotal.Add(1)
			e.logger.Error("failed to weave charm on pod",
				zap.String("pod", pod.Name),
				zap.Error(err),
			)
			failed++
			continue
		}

		e.targetsAffected.Add(1)
		e.operationsTotal.Add(1)
		e.logger.Info("charm takes hold", zap.String("pod", pod.Name))
		charmed++
	}

	return charmed, failed
}

func (e *CharmEye) charmPod(ctx context.Context, namespace, podName, containerName string, cfg *Config) error {
	ephContainer := buildNetshootContainer(containerName)

	err := e.ephemeralContainers.AddEphemeralContainer(ctx, namespace, podName, ephContainer)
	if err != nil {
		return fmt.Errorf("inject netshoot sidecar: %w", err)
	}

	applyCommand := buildApplyCommand(cfg, cfg.Interface)

	err = e.ephemeralContainers.ExecInContainer(ctx, namespace, podName, containerName, applyCommand)
	if err != nil {
		return fmt.Errorf("apply tc netem rules: %w", err)
	}

	e.trackCharmed(namespace, podName, containerName, cfg.Interface)

	return nil
}

func (e *CharmEye) Pause(ctx context.Context) error {
	return e.removeAllCharms(ctx)
}

func (e *CharmEye) Close(ctx context.Context) error {
	defer e.active.Store(false)
	return e.removeAllCharms(ctx)
}

func (e *CharmEye) Observe() eyes.Metrics {
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

func (e *CharmEye) removeAllCharms(ctx context.Context) error {
	e.mu.Lock()
	pods := make([]charmedPod, len(e.charmed))
	copy(pods, e.charmed)
	e.mu.Unlock()

	for _, pod := range pods {
		cleanupCommand := buildCleanupCommand(pod.iface)
		err := e.ephemeralContainers.ExecInContainer(ctx, pod.namespace, pod.podName, pod.containerName, cleanupCommand)
		if err != nil {
			e.logger.Warn("failed to remove charm from pod — manual cleanup may be needed",
				zap.String("pod", pod.podName),
				zap.String("namespace", pod.namespace),
				zap.Error(err),
			)
		}
	}

	return nil
}

func (e *CharmEye) trackCharmed(namespace, podName, containerName, iface string) {
	e.mu.Lock()
	e.charmed = append(e.charmed, charmedPod{
		namespace:     namespace,
		podName:       podName,
		containerName: containerName,
		iface:         iface,
	})
	e.mu.Unlock()
}

func (e *CharmEye) addTruth(truth string) {
	e.mu.Lock()
	e.truthsRevealed = append(e.truthsRevealed, truth)
	e.mu.Unlock()
}

func buildNetshootContainer(name string) corev1.EphemeralContainer {
	return corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:    name,
			Image:   netshootImage,
			Command: []string{"sleep", "infinity"},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"NET_ADMIN"},
				},
			},
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

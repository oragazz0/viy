package disintegration

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/oragazz0/viy/internal/k8s"
	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

func init() {
	eyes.Register("disintegration", func() eyes.Eye {
		return &Eye{}
	})
}

// Eye reveals pod auto-recovery and orchestration health.
type Eye struct {
	podManager      k8s.PodManager
	logger          *zap.Logger
	targetsAffected atomic.Int64
	operationsTotal atomic.Int64
	errorsTotal     atomic.Int64
	truthsRevealed  []string
	lastExecution   atomic.Int64
	active          atomic.Bool
}

// NewEye creates a DisintegrationEye with dependencies.
func NewEye(podManager k8s.PodManager, logger *zap.Logger) *Eye {
	return &Eye{
		podManager: podManager,
		logger:     logger,
	}
}

func (e *Eye) Name() string {
	return "disintegration"
}

func (e *Eye) Description() string {
	return "Reveals pod auto-recovery and orchestration health"
}

func (e *Eye) Validate(config eyes.EyeConfig) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("%w: expected *disintegration.Config", viyerrors.ErrInvalidConfiguration)
	}

	return cfg.Validate()
}

func (e *Eye) Unveil(ctx context.Context, target eyes.Target, config eyes.EyeConfig) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("%w: expected *disintegration.Config", viyerrors.ErrInvalidConfiguration)
	}

	e.active.Store(true)
	defer e.active.Store(false)

	pods, err := e.podManager.GetPods(ctx, target.Namespace, target.Selector)
	if err != nil {
		return fmt.Errorf("resolving targets: %w", err)
	}

	killCount := cfg.PodKillCount
	if cfg.PodKillPercentage > 0 {
		killCount = len(pods) * cfg.PodKillPercentage / 100
		if killCount == 0 {
			killCount = 1
		}
	}

	if killCount > len(pods) {
		return fmt.Errorf("%w: want %d but only %d available",
			viyerrors.ErrInsufficientTargets, killCount, len(pods))
	}

	selected := selectPods(pods, killCount, cfg.Strategy)
	gracePeriod := int64(cfg.GracePeriod.Seconds())

	for index, pod := range selected {
		if index > 0 && cfg.Interval > 0 {
			e.logger.Info("waiting before next revelation",
				zap.Duration("interval", cfg.Interval),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.Interval):
			}
		}

		e.logger.Info("unveiling pod",
			zap.String("pod", pod.Name),
			zap.String("namespace", pod.Namespace),
		)

		if err := e.podManager.DeletePod(ctx, pod.Namespace, pod.Name, gracePeriod); err != nil {
			e.errorsTotal.Add(1)
			return fmt.Errorf("unveiling pod %s: %w", pod.Name, err)
		}

		e.operationsTotal.Add(1)
		e.targetsAffected.Add(1)
		e.lastExecution.Store(time.Now().UnixNano())
	}

	e.truthsRevealed = append(e.truthsRevealed,
		fmt.Sprintf("Revealed %d pods in %s/%s", killCount, target.Namespace, target.Name))

	return nil
}

func (e *Eye) Pause(_ context.Context) error {
	e.active.Store(false)
	return nil
}

func (e *Eye) Close(_ context.Context) error {
	e.active.Store(false)
	return nil
}

func (e *Eye) Observe() eyes.Metrics {
	return eyes.Metrics{
		TargetsAffected:   int(e.targetsAffected.Load()),
		OperationsTotal:   e.operationsTotal.Load(),
		ErrorsTotal:       e.errorsTotal.Load(),
		TruthsRevealed:    e.truthsRevealed,
		LastExecutionTime: time.Unix(0, e.lastExecution.Load()),
		IsActive:          e.active.Load(),
	}
}

func selectPods(pods []corev1.Pod, count int, strategy string) []corev1.Pod {
	if strategy == "sequential" {
		return pods[:count]
	}

	shuffled := make([]corev1.Pod, len(pods))
	copy(shuffled, pods)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

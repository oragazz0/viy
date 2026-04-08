package cli

import (
	"context"
	"fmt"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oragazz0/viy/internal/eyes/disintegration"
	"github.com/oragazz0/viy/internal/k8s"
	"github.com/oragazz0/viy/internal/observability"
	"github.com/oragazz0/viy/internal/orchestrator"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/eyes"
)

var protectedNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

func newUnveilCommand() *cobra.Command {
	var (
		eyeName     string
		target      string
		namespace   string
		selector    string
		duration    time.Duration
		blastRadius string
		configStr   string
		dream       bool
		minHealthy  int
	)

	command := &cobra.Command{
		Use:   "unveil",
		Short: "Open an eye — start a chaos experiment",
		Long: color.MagentaString("🔮 ") +
			"Unveil hidden weaknesses by opening an eye of Viy.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUnveil(eyeName, target, namespace, selector, duration, blastRadius, configStr, dream, minHealthy)
		},
	}

	command.Flags().StringVar(&eyeName, "eye", "", "Eye to open (required)")
	command.Flags().StringVar(&target, "target", "", "Target resource, e.g. deployment/nginx (required)")
	command.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	command.Flags().StringVar(&selector, "selector", "", "Label selector to filter pods, e.g. version=v2")
	command.Flags().DurationVar(&duration, "duration", 5*time.Minute, "Revelation duration")
	command.Flags().StringVar(&blastRadius, "blast-radius", "30%", "Max %% of targets to affect")
	command.Flags().StringVar(&configStr, "config", "", "Eye-specific config (key=value pairs)")
	command.Flags().BoolVar(&dream, "dream", false, "Dry-run mode")
	command.Flags().IntVar(&minHealthy, "min-healthy", 1, "Minimum healthy replicas to preserve")

	if err := command.MarkFlagRequired("eye"); err != nil {
		panic(fmt.Sprintf("marking eye flag required: %v", err))
	}

	if err := command.MarkFlagRequired("target"); err != nil {
		panic(fmt.Sprintf("marking target flag required: %v", err))
	}

	return command
}

func runUnveil(eyeName, target, namespace, selector string, duration time.Duration, blastRadius, configStr string, dream bool, minHealthy int) error {
	if protectedNamespaces[namespace] {
		return fmt.Errorf("namespace %q is protected — chaos experiments are not allowed in system namespaces", namespace)
	}

	logger, err := observability.NewLogger(logLevel)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	k8sClient, err := k8s.NewClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("connecting to kubernetes: %w", err)
	}

	store, err := state.NewStore()
	if err != nil {
		return fmt.Errorf("initializing state: %w", err)
	}

	blastPct, err := parsePercentage(blastRadius)
	if err != nil {
		return fmt.Errorf("invalid blast-radius: %w", err)
	}

	eyeConfig := buildDisintegrationConfig(configStr)
	resolver := k8s.NewResolver(k8sClient)
	orch := orchestrator.NewOrchestrator(k8sClient, resolver, store, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Println()
	color.Magenta("👁️  Viy awakens from slumber...")
	fmt.Println()

	return orch.Run(ctx, orchestrator.RunConfig{
		EyeName: eyeName,
		Target: eyes.Target{
			Kind:      parseKindFromTarget(target),
			Name:      parseNameFromTarget(target),
			Namespace: namespace,
			Selector:  selector,
		},
		EyeConfig:          eyeConfig,
		Duration:           duration,
		BlastRadius:        blastPct,
		MinHealthyReplicas: minHealthy,
		DryRun:             dream,
	})
}

func parsePercentage(value string) (int, error) {
	cleaned := strings.TrimSuffix(value, "%")

	parsed, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0, err
	}

	if parsed < 1 || parsed > 100 {
		return 0, fmt.Errorf("blast radius must be between 1%% and 100%%, got %d%%", parsed)
	}

	return parsed, nil
}

func parseKindFromTarget(target string) string {
	parts := strings.SplitN(target, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}

	return "pod"
}

func parseNameFromTarget(target string) string {
	parts := strings.SplitN(target, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return target
}

func buildDisintegrationConfig(configStr string) *disintegration.Config {
	config := &disintegration.Config{
		PodKillCount: 1,
		Strategy:     "random",
	}

	if configStr == "" {
		return config
	}

	pairs := strings.Split(configStr, ",")
	validStrategies := map[string]bool{"random": true, "sequential": true}

	for _, pair := range pairs {
		keyValue := strings.SplitN(pair, "=", 2)
		if len(keyValue) != 2 {
			continue
		}

		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])

		switch key {
		case "podKillCount":
			if count, err := strconv.Atoi(value); err == nil && count > 0 {
				config.PodKillCount = count
				config.PodKillPercentage = 0
			}
		case "podKillPercentage":
			if pct, err := strconv.Atoi(strings.TrimSuffix(value, "%")); err == nil && pct > 0 {
				config.PodKillPercentage = pct
				config.PodKillCount = 0
			}
		case "interval":
			if dur, err := time.ParseDuration(value); err == nil {
				config.Interval = dur
			}
		case "strategy":
			if validStrategies[value] {
				config.Strategy = value
			}
		case "gracePeriod":
			if dur, err := time.ParseDuration(value); err == nil {
				config.GracePeriod = dur
			}
		}
	}

	return config
}

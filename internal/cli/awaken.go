package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oragazz0/viy/internal/k8s"
	"github.com/oragazz0/viy/internal/observability"
	"github.com/oragazz0/viy/internal/orchestrator"
	"github.com/oragazz0/viy/internal/state"
	"github.com/oragazz0/viy/pkg/config"
	"github.com/oragazz0/viy/pkg/eyes"
)

func newAwakenCommand() *cobra.Command {
	var (
		filePath string
		dream    bool
	)

	command := &cobra.Command{
		Use:   "awaken",
		Short: "Open many eyes at once — start a multi-eye chaos experiment from YAML",
		Long: color.MagentaString("🔮 ") +
			"Awaken multiple Eyes of Viy simultaneously from an experiment YAML.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAwaken(filePath, dream)
		},
	}

	command.Flags().StringVar(&filePath, "file", "", "Path to experiment YAML (required)")
	command.Flags().BoolVar(&dream, "dream", false, "Dry-run mode")

	if err := command.MarkFlagRequired("file"); err != nil {
		panic(fmt.Sprintf("marking file flag required: %v", err))
	}

	return command
}

func runAwaken(filePath string, dream bool) error {
	experiment, err := config.Load(filePath)
	if err != nil {
		return err
	}

	if err := experiment.Validate(); err != nil {
		return err
	}

	if err := ensureAwakenNamespacesAllowed(experiment); err != nil {
		return err
	}

	multiConfig, err := buildMultiConfig(experiment, dream)
	if err != nil {
		return err
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

	resolver := k8s.NewResolver(k8sClient)
	orch := orchestrator.NewOrchestrator(k8sClient, resolver, store, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Println()
	color.Magenta("👁️👁️  Viy's many eyes open in unison...")
	fmt.Println()

	return orch.RunMulti(ctx, multiConfig)
}

func ensureAwakenNamespacesAllowed(experiment *config.Experiment) error {
	for _, eye := range experiment.Spec.Eyes {
		if err := ensureNamespaceAllowed(eye.Target.Namespace); err != nil {
			return fmt.Errorf("eye %q: %w", eye.Name, err)
		}
	}

	return nil
}

func buildMultiConfig(experiment *config.Experiment, dream bool) (orchestrator.MultiConfig, error) {
	specs := make([]orchestrator.EyeRunSpec, 0, len(experiment.Spec.Eyes))

	for _, eyeSpec := range experiment.Spec.Eyes {
		eyeConfig, err := config.DecodeConfig(eyeSpec.Name, eyeSpec.Config)
		if err != nil {
			return orchestrator.MultiConfig{}, err
		}

		specs = append(specs, orchestrator.EyeRunSpec{
			Name:     eyeSpec.Name,
			Target:   buildTargetFromSpec(eyeSpec.Target),
			Config:   eyeConfig,
			Duration: eyeSpec.Duration.ToStd(),
		})
	}

	return orchestrator.MultiConfig{
		ExperimentName:  experiment.Metadata.Name,
		Duration:        experiment.Spec.Duration.ToStd(),
		FailurePolicy:   orchestrator.FailurePolicy(experiment.Spec.FailurePolicy.Resolve()),
		StaggerInterval: experiment.Spec.StaggerInterval.ToStd(),
		StrictIsolation: experiment.Spec.StrictIsolation,
		BlastRadius:     experiment.Spec.Safety.MaxBlastRadius,
		MinHealthy:      experiment.Spec.Safety.MinHealthyReplicas,
		DryRun:          dream,
		Eyes:            specs,
	}, nil
}

func buildTargetFromSpec(spec config.TargetSpec) eyes.Target {
	return eyes.Target{
		Kind:      spec.Kind,
		Name:      spec.Name,
		Namespace: spec.Namespace,
		Selector:  spec.Selector,
	}
}

package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDreamCommand() *cobra.Command {
	var (
		eyeName     string
		target      string
		namespace   string
		blastRadius string
		configStr   string
		minHealthy  int
	)

	command := &cobra.Command{
		Use:   "dream",
		Short: "Dream of revelation — dry-run without executing chaos",
		Long: color.MagentaString("🔮 ") +
			"Viy dreams of chaos without executing. Shows what would happen.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUnveil(eyeName, target, namespace, 0, blastRadius, configStr, true, minHealthy)
		},
	}

	command.Flags().StringVar(&eyeName, "eye", "", "Eye to open (required)")
	command.Flags().StringVar(&target, "target", "", "Target resource (required)")
	command.Flags().StringVar(&namespace, "namespace", "default", "Kubernetes namespace")
	command.Flags().StringVar(&blastRadius, "blast-radius", "30%", "Max %% of targets to affect")
	command.Flags().StringVar(&configStr, "config", "", "Eye-specific config (key=value pairs)")
	command.Flags().IntVar(&minHealthy, "min-healthy", 1, "Minimum healthy replicas to preserve")

	if err := command.MarkFlagRequired("eye"); err != nil {
		panic(fmt.Sprintf("marking eye flag required: %v", err))
	}

	if err := command.MarkFlagRequired("target"); err != nil {
		panic(fmt.Sprintf("marking target flag required: %v", err))
	}

	return command
}

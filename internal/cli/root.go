package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const banner = `
══════════════════════════════════════════════════════════════
  ⬡       ⬡       ⬡
   ╲      │      ╱
     ▄████████████▄
    █░░╱ ▓▒◉▒▓ ╲░░█       V I Y
     ▀████████████▀
   ╱      │      ╲        Kubernetes Chaos Engineering Toolkit
  ⬡       ⬡       ⬡      	"Omniscient chaos, unveiled"
══════════════════════════════════════════════════════════════
`

var (
	logLevel   string
	output     string
	kubeconfig string
)

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "viy",
		Short: "Kubernetes Chaos Engineering Toolkit — Omniscient chaos, unveiled",
		Long:  color.MagentaString(banner),
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level: debug|info|warn|error")
	root.PersistentFlags().StringVar(&output, "output", "text", "Output format: text|json")
	root.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")

	root.AddCommand(
		newUnveilCommand(),
		newDreamCommand(),
		newSlumberCommand(),
		newVisionCommand(),
		newVersionCommand(),
	)

	return root
}

// Execute runs the root CLI command.
func Execute() error {
	root := newRootCommand()

	if err := root.Execute(); err != nil {
		return fmt.Errorf("cli: %w", err)
	}

	return nil
}

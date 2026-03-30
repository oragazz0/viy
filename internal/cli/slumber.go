package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oragazz0/viy/internal/state"
)

func newSlumberCommand() *cobra.Command {
	var (
		all          bool
		experimentID string
		force        bool
	)

	command := &cobra.Command{
		Use:   "slumber",
		Short: "Close the eyes — stop experiments",
		Long: color.MagentaString("😴 ") +
			"Close all eyes and stop active experiments.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSlumber(all, experimentID, force)
		},
	}

	command.Flags().BoolVar(&all, "all", false, "Close all eyes (stop all experiments)")
	command.Flags().StringVar(&experimentID, "experiment", "", "Stop a specific experiment")
	command.Flags().BoolVar(&force, "force", false, "Force stop without cleanup")

	return command
}

// TODO(security): slumber only updates the state file — it does not cancel
// running experiments in other processes. Implement cross-process cancellation
// via PID files or a file-watch mechanism.
func runSlumber(all bool, experimentID string, _ bool) error {
	store, err := state.NewStore()
	if err != nil {
		return err
	}

	experiments, err := store.Load()
	if err != nil {
		return err
	}

	updated := false

	for index := range experiments {
		isTarget := all || experiments[index].ID == experimentID
		isActive := experiments[index].Status == state.StatusUnveiling

		if !isTarget || !isActive {
			continue
		}

		experiments[index].Status = state.StatusRevealed
		updated = true
		color.Yellow("😴 Closing eyes on experiment %s", experiments[index].ID)
	}

	if !updated {
		fmt.Println("No active experiments to close.")
		return nil
	}

	return store.Save(experiments)
}

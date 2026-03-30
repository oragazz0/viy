package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oragazz0/viy/internal/state"
)

func newVisionCommand() *cobra.Command {
	var showAll bool

	command := &cobra.Command{
		Use:   "vision",
		Short: "See active experiments",
		Long: color.MagentaString("👁️ ") +
			"View all active and recent experiments.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runVision(showAll)
		},
	}

	command.Flags().BoolVar(&showAll, "all", false, "Show completed experiments too")

	return command
}

func runVision(showAll bool) error {
	store, err := state.NewStore()
	if err != nil {
		return err
	}

	experiments, err := store.Load()
	if err != nil {
		return err
	}

	if len(experiments) == 0 {
		fmt.Println("No experiments found. Viy sleeps.")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer func() { _ = writer.Flush() }()

	fmt.Fprintln(writer, "ID\tEYES\tSTATUS\tTARGET\tSTARTED")

	for _, experiment := range experiments {
		if !showAll && experiment.Status != state.StatusUnveiling {
			continue
		}

		age := time.Since(experiment.StartTime).Truncate(time.Second)
		eyeList := fmt.Sprintf("%v", experiment.Eyes)

		fmt.Fprintf(writer, "%s\t%s\t%s\t%s/%s\t%s ago\n",
			experiment.ID,
			eyeList,
			string(experiment.Status),
			experiment.Namespace,
			experiment.Target,
			age,
		)
	}

	return nil
}

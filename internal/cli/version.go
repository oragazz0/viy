package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/oragazz0/viy/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Viy version",
		Run: func(_ *cobra.Command, _ []string) {
			color.Magenta("👁️  Viy")
			fmt.Printf("  Version: %s\n", version.Version)
			fmt.Printf("  Commit:  %s\n", version.Commit)
			fmt.Printf("  Built:   %s\n", version.Date)
		},
	}
}

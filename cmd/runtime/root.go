package runtime

import (
	"github.com/spf13/cobra"
)

// RuntimeCmd represents the base "runtime" command.
var RuntimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Manage mule applications",
	Long:  `Perform operations related to Anypoint runtime, such as listing, creating, and deleting applications.`,
}

func init() {
	// Add subcommands to RuntimeCmd.
	RuntimeCmd.AddCommand(listAppsCmd)
	RuntimeCmd.AddCommand(monitorCmd)

	// Here you can add persistent flags for the exchange group if needed.
	// For example, a flag to specify an environment or organization ID if they are common to all subcommands.
}

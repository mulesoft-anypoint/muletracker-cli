package apim

import (
	"github.com/spf13/cobra"
)

// ApimCmd is the root command for API Management operations.
var ApimCmd = &cobra.Command{
	Use:   "apim",
	Short: "Manage API Manager instances",
	Long:  "Perform operations related to API Manager, such as listing instances with filters and exporting data.",
}

func init() {
	// Register subcommands.
	ApimCmd.AddCommand(listApiCmd)
	ApimCmd.AddCommand(deleteCmd)
	// Later add create, delete, etc.
}

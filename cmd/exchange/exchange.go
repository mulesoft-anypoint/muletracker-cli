package exchange

import (
	"github.com/spf13/cobra"
)

// ExchangeCmd represents the base "exchange" command.
var ExchangeCmd = &cobra.Command{
	Use:   "exchange",
	Short: "Manage Exchange applications",
	Long:  `Perform operations related to Anypoint Exchange, such as listing, creating, and deleting Exchange applications.`,
}

func init() {
	// Add subcommands to ExchangeCmd.
	ExchangeCmd.AddCommand(listCmd)
	ExchangeCmd.AddCommand(createCmd)
	ExchangeCmd.AddCommand(deleteCmd)

	// Here you can add persistent flags for the exchange group if needed.
	// For example, a flag to specify an environment or organization ID if they are common to all subcommands.
}
